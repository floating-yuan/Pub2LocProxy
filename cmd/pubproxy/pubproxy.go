package pubproxy

import (
	"bufio"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/floating-yuan/pub2locproxy/internal/auth"
	"github.com/floating-yuan/pub2locproxy/internal/config"
	"github.com/spf13/cobra"
)

var (
	addrForInput  string
	addrForClient string
)

const (
	DefaultAddrForInput  = ":16350"
	DefaultAddrForClient = "0.0.0.0:9900"
)

var serverCh = make(chan net.Conn)

var routeCliRegister = map[string]net.Conn{}

var PubProxyCmd = &cobra.Command{
	Use:   "pubproxy",
	Short: "pubproxy",
	Long:  `pubproxy`,
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

func init() {
	PubProxyCmd.PersistentFlags().StringVarP(&addrForInput, "input", "i", "", "addrForInput")
	PubProxyCmd.PersistentFlags().StringVarP(&addrForClient, "client", "c", "", "addrForClient")
}

func run() {

	cnf := config.GetConfig()

	ia := addrForInput
	if ia == "" {
		ia = DefaultAddrForInput
	}

	if cnf.Pubproxy != nil || cnf.Pubproxy.InputAddr != "" {
		ia = cnf.Pubproxy.InputAddr
	}

	lnServer, err := net.Listen("tcp", ia)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("addrForInput: ", ia)

	ca := addrForClient
	if ca == "" {
		ca = DefaultAddrForClient
	}

	if cnf.Pubproxy != nil || cnf.Pubproxy.ServerAddr != "" {
		ca = cnf.Pubproxy.ServerAddr
	}

	lnClients, err := net.Listen("tcp", ca)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("addrForClient: ", ca)

	//listen for input connection
	go func() {
		for {
			conn, err := lnServer.Accept()
			log.Println("a input request has been connected!")
			if err != nil {
				// handle errors
			}
			serverCh <- conn
		}
	}()

	//handle new forward client connection

	go func() {
		for {
			var forwardCli net.Conn
			//outside  --input--> nat gw
			forwardCli, err = lnClients.Accept()
			log.Println("a client has been connected!")
			if err != nil {
				// handle errors
			}

			log.Println("authToken read begin")
			authToken, err := bufio.NewReader(forwardCli).ReadString('\n')
			if err != nil {
				log.Println("auth token read err:", err)
				continue
			}

			var authErr error
			valid, svcUser, err := auth.Authenticate(strings.TrimSuffix(authToken, "\n"))
			_ = svcUser
			if err != nil {
				authErr = err
			} else if !valid {
				authErr = errors.New("auth token invalid")
			}

			if authErr != nil {
				log.Println("authErr write begin")
				writeN, writeErr := forwardCli.Write([]byte(err.Error() + "\n"))
				_ = writeN
				if writeErr != nil {
					log.Println("write auth err failed :", writeErr)
				}
				forwardCli.Close()
				continue
			}

			log.Println("AuthOK write begin")
			writeN, writeErr := forwardCli.Write([]byte("AuthOK\n"))
			_ = writeN
			if writeErr != nil {
				log.Println("write AuthOK failed :", writeErr)
				continue
			}

			var route string
			log.Println("register route read begin")
			switch 2 {
			case 1:
				routeBytes := make([]byte, 120)
				forwardCli.Read(routeBytes)

			case 2:
				log.Println("register route read case 2 begin")
				route, err = bufio.NewReader(forwardCli).ReadString('\n')
				if err != nil {
					log.Println("register route read err:", err)
					continue
				}

				route = strings.TrimSuffix(route, "\n")
			}

			log.Println("register route:", route)

			if _, ok := routeCliRegister[route]; ok {
				writeN, writeErr := forwardCli.Write([]byte("this route has already been registed" + "\n"))
				_ = writeN
				if writeErr != nil {
					log.Println("write auth err failed :", writeErr)
				}
				forwardCli.Close()
				continue
			}

			routeCliRegister[route] = forwardCli

			log.Println("RouteRegistedOK write begin")
			writeN, writeErr = forwardCli.Write([]byte("RouteRegistedOK\n"))
			_ = writeN
			if writeErr != nil {
				log.Println("write RouteRegistedOK failed :", writeErr)
				continue
			}

			go checkConnectionStatus(route, forwardCli)

			//inputConn := <-serverCh
			//go io.Copy(inputConn, forwardCli)
			//io.Copy(forwardCli, inputConn)
		}
	}()

	//handle new input connection
	for {
		select {
		case inputConn := <-serverCh:

			var matchedConn net.Conn
			var matchedRoute string
			var readRequest *http.Request
			log.Println("serverCh new input conn!")

			if true {
				log.Println("read request from input conn")
				reader := bufio.NewReader(inputConn)
				var readErr error
				readRequest, readErr = http.ReadRequest(reader)

				if readErr == io.EOF {
					log.Println("readErr == io.EOF")
				} else {
					for routeKey, connVal := range routeCliRegister {
						if strings.HasPrefix(readRequest.URL.Path, routeKey) {
							matchedConn = connVal
							matchedRoute = routeKey
							break
						}
					}
				}
			}

			if matchedConn == nil {
				errRsp := http.Response{}
				errRsp.StatusCode = http.StatusNotFound
				errBodyContent := "HasNoMatchedRoute"
				reader := strings.NewReader(errBodyContent)
				readCloser := ioutil.NopCloser(reader)
				errRsp.Body = readCloser
				errRsp.ContentLength = int64(len(errBodyContent))
				err = errRsp.Write(inputConn)
				if err != nil {
					log.Println("rsp.Write(inputConn) err:", err)
				}

				continue
			}

			//send input request to the matched forward client connection
			log.Println("read request send to route:", matchedRoute, "; forward conn:", matchedConn.RemoteAddr().String())
			readRequest.Write(matchedConn)
			go func() {

				//forwardCli  --> nat gw inputConn
				log.Println("matchedConn  --> inputConn begin")
				if true {
					reader := bufio.NewReader(matchedConn)
					log.Println("ReadResponse begin")
					rsp, readErr := http.ReadResponse(reader, nil)
					if readErr != nil {
						log.Println("readErr:", readErr)
						return
					}

					log.Println("ReadResponse over, write into inputConn begin")
					inputConn.SetWriteDeadline(time.Now().Add(3 * time.Second))
					rsp.Write(inputConn)

					log.Println("write into inputConn over")
					inputConn.Close()
				}

			}()

		}
	}

}

func checkConnectionStatus(route string, conn net.Conn) {
	for {
		// Check if the connection is closed by writing to it
		//fmt.Println("checkConnectionStatus begin:", "; route:", route, "; conn:", conn.RemoteAddr().String())
		_, err := conn.Write([]byte{})
		if err != nil {
			log.Println("Connection closed err:", err, "; route:", route, "; conn:", conn.RemoteAddr().String())

			conn.Close()
			delete(routeCliRegister, route)
			return
		}

		// Sleep for a brief period before checking again
		time.Sleep(2 * time.Second)
	}
}
