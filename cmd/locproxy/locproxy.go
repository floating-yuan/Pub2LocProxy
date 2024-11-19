package locproxy

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/floating-yuan/pub2locproxy/internal/auth"
	"github.com/floating-yuan/pub2locproxy/internal/config"
	"github.com/spf13/cobra"
)

var (
	forwardAddr     string
	natGwServerAddr string
)

const (
	DefaultNatGwServerAddr = "127.0.0.1:9900"
	DefaultForwardAddr     = "127.0.0.1:9910"
)

var LocProxyCmd = &cobra.Command{
	Use:   "locproxy",
	Short: "locproxy",
	Long:  `locproxy`,
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

func init() {
	LocProxyCmd.PersistentFlags().StringVarP(&forwardAddr, "forward", "f", "", "target address for forward")
	LocProxyCmd.PersistentFlags().StringVarP(&natGwServerAddr, "server", "s", "", "nat gw server address")
}

func forwardConnection4(proxyServerConn net.Conn, remoteAddr string) {
	var readFullMsg string
	for {
		log.Println("proxyServerConn ReadRequest full:", readFullMsg)
		reader := bufio.NewReader(proxyServerConn)
		request, readErr := http.ReadRequest(reader)

		if readErr == io.EOF {
			log.Println("readErr == io.EOF")
			break
		}

		request.URL.Scheme = "http"
		request.URL.Host = remoteAddr
		request.RequestURI = ""
		log.Println("proxyServerConn ReadString:", request, "; err:", readErr)

		log.Println("http.DefaultClient.Do(request):")
		rsp, err := http.DefaultClient.Do(request)
		if err != nil {
			log.Println("http.DefaultClient.Do(request) err:", err)

			errRsp := http.Response{}
			errRsp.StatusCode = http.StatusServiceUnavailable
			errBodyContent := "ServiceUnavailable"
			reader := strings.NewReader(errBodyContent)
			readCloser := ioutil.NopCloser(reader)
			errRsp.Body = readCloser
			errRsp.ContentLength = int64(len(errBodyContent))
			err = errRsp.Write(proxyServerConn)
			if err != nil {
				log.Println("rsp.Write(proxyServerConn) err:", err)
			}
			continue
		}

		err = rsp.Write(proxyServerConn)
		if err != nil {
			log.Println("rsp.Write(proxyServerConn) err:", err)
		}
	}

}

func forwardConnection3(proxyServerConn net.Conn, remoteAddr string) {

	var readFullMsg string
	for {
		log.Println("proxyServerConn ReadString full:", readFullMsg)
		readMsg, readErr := bufio.NewReader(proxyServerConn).ReadString('\n')

		log.Println("proxyServerConn ReadString:", readMsg, "; err:", readErr)
		if readErr == io.EOF {
			log.Println("readErr == io.EOF")
			break
		}

		readFullMsg += readMsg

	}

	remoteConn, err := net.Dial("tcp", remoteAddr)
	if err != nil {
		log.Printf("Failed to connect to remote server: %v", err)
		proxyServerConn.Close()
		return
	}

	log.Println("forwardAddr: ", remoteAddr)

	go func() {
		var rbuf []byte
		log.Println("remoteConn.Read")
		readN, readErr := remoteConn.Read(rbuf)
		log.Println("remoteConn.Read , len:", readN, "; msg:", string(rbuf), "; write err:", readErr)

		log.Println("proxyServerConn.Write")
		writeN, writeErr := proxyServerConn.Write(rbuf)
		log.Println("proxyServerConn.Write, len:", writeN, "; msg:", string(rbuf), "; write err:", writeErr)
		remoteConn.Close()
	}()

	log.Println("rbuf  --> remoteConn write")
	writeN, writeErr := remoteConn.Write([]byte(readFullMsg))
	log.Println("rbuf  --> remoteConn write , len:", writeN, "; msg:", string(readFullMsg), "; write err:", writeErr)

}

func forwardConnection2(proxyServerConn net.Conn, remoteAddr string) {

	for {
		var rbuf []byte
		log.Printf("forwardConnection waiting for new messages from nat gw server")
		message, _ := bufio.NewReader(proxyServerConn).ReadString('\n')
		fmt.Print("Message from server: " + message)
		readN, readErr := proxyServerConn.Read(rbuf)
		log.Println("forwardConnection got a new messages , len:", readN, "; msg:", string(rbuf), "; write err:", readErr)

		remoteConn, err := net.Dial("tcp", remoteAddr)
		if err != nil {
			log.Printf("Failed to connect to remote server: %v", err)
			proxyServerConn.Close()
			return
		}

		log.Println("forwardAddr: ", remoteAddr)

		go func() {
			var rbuf []byte
			log.Printf("remoteConn.Read")
			readN, readErr := remoteConn.Read(rbuf)
			log.Println("remoteConn.Read , len:", readN, "; msg:", string(rbuf), "; write err:", readErr)

			log.Printf("proxyServerConn.Write")
			writeN, writeErr := proxyServerConn.Write(rbuf)
			log.Println("proxyServerConn.Write, len:", writeN, "; msg:", string(rbuf), "; write err:", writeErr)
			remoteConn.Close()
		}()

		log.Printf("rbuf  --> remoteConn write")
		writeN, writeErr := remoteConn.Write(rbuf)
		log.Println("rbuf  --> remoteConn write , len:", writeN, "; msg:", string(rbuf), "; write err:", writeErr)
	}

}

func forwardConnection(proxyServerConn net.Conn, remoteAddr string) {
	log.Printf("forwardConnection begin")
	remoteConn, err := net.Dial("tcp", remoteAddr)
	if err != nil {
		log.Printf("Failed to connect to remote server: %v", err)
		proxyServerConn.Close()
		return
	}

	log.Println("forwardAddr: ", remoteAddr)

	go func() {
		log.Printf("proxyServerConn --> remoteConn begin")
		copyN, copyErr := io.Copy(remoteConn, proxyServerConn)
		log.Println("proxyServerConn --> remoteConn ret: ", copyN, copyErr)
	}()

	log.Printf("remoteConn --> proxyServerConn begin")
	copyN, copyErr := io.Copy(proxyServerConn, remoteConn)
	log.Println("remoteConn --> proxyServerConn ret: ", copyN, copyErr)

	log.Printf("forwardConnection over")

	//proxyServerConn.Close()
	//remoteConn.Close()
}

func run() {

	cnf := config.GetConfig()

	if cnf == nil || cnf.Locproxy == nil {
		log.Println("config of Forward empty!")
		return
	}

	if cnf.Locproxy.User == nil {
		log.Println("Forward.User empty!")
		return
	}

	if cnf.Locproxy.RegisterRoute == "" {
		log.Println("cnf.Forward.RegisterRoute empty!")
	}

	sa := natGwServerAddr
	if sa == "" {
		sa = DefaultNatGwServerAddr
	}

	if cnf.Locproxy.ServerAddr != "" {
		sa = cnf.Locproxy.ServerAddr
	}

	log.Println("natGwServerAddr: ", sa)
	proxyServerConn, err := net.Dial("tcp", sa)
	if err != nil {
		log.Printf("Failed to connect to nat gw server: %v", err)
		return
	}

	defer proxyServerConn.Close()

	//step1.1:auth
	authToken, err := auth.GenerateToken(cnf.Locproxy.User.Secret, cnf.Locproxy.User.AccessKey)
	if err != nil {
		log.Println("auth.GenerateToken err: ", err)
		return
	}

	log.Println("authToken write begin")
	authWN, authWErr := proxyServerConn.Write([]byte(authToken + "\n"))
	if authWErr != nil {
		log.Println("authWErr: ", authWErr)
		return
	}
	_ = authWN

	//step1.2:auth confirm
	log.Println("authConfirm read begin")
	authConfirm, err := bufio.NewReader(proxyServerConn).ReadString('\n')
	if err != nil {
		log.Println("authConfirm read err:", err)
		return
	}

	if authConfirm != "AuthOK\n" {
		log.Println("Auth is not OK:", authConfirm)
		return
	} else {
		log.Println("AuthOK")
	}

	//step2.1:register route
	registerRoutePattern := cnf.Locproxy.RegisterRoute
	log.Println("registerRoutePattern write begin")
	regN, regErr := proxyServerConn.Write([]byte(registerRoutePattern + "\n"))
	if regErr != nil {
		log.Println("registerRoutePattern err: ", regErr)
		return
	}
	_ = regN

	//step2.2:register confirm
	log.Println("registerConfirm read begin")
	registerConfirm, err := bufio.NewReader(proxyServerConn).ReadString('\n')
	if err != nil {
		log.Println("registerConfirm read err:", err)
		return
	}

	if registerConfirm != "RouteRegistedOK\n" {
		log.Println("Route Register is not OK:", registerConfirm)
		return
	} else {
		log.Println("RouteRegistedOK")
	}

	fa := forwardAddr
	if fa == "" {
		fa = DefaultForwardAddr
	}

	if cnf.Locproxy.Forward != "" {
		fa = cnf.Locproxy.Forward
	}

	log.Println("forwardConnection4 begin: ", fa)
	forwardConnection4(proxyServerConn, fa)
}
