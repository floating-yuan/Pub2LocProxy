package auth

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/floating-yuan/pub2locproxy/internal/config"
	"github.com/golang-jwt/jwt"
)

type ServiceUser struct {
	Id        int
	Name      string
	AccessKey string
	Secret    string
}

func getIdentityByAccessKey(akVal string) (svcUser *ServiceUser, err error) {

	cnf := config.GetConfig()

	if cnf == nil || cnf.Pubproxy == nil {
		err = errors.New("server config empty")
		return
	}

	if cnf.Pubproxy.UserMap == nil {
		err = errors.New("Server.UserMap empty")
		return
	}

	if u, ok := cnf.Pubproxy.UserMap[akVal]; ok {
		svcUser = &ServiceUser{
			Id:        1,
			AccessKey: u.AccessKey,
			Secret:    u.Secret,
		}
	} else {
		err = errors.New("user not found")
	}

	return
}

func Authenticate(authToken string) (valid bool, svcUser *ServiceUser, err error) {
	//BOM，deal with Byte Order Mark problem
	authToken = string(bytes.TrimPrefix([]byte(authToken), []byte("\xef\xbb\xbf"))) // Or []byte{239, 187, 191}

	var token *jwt.Token
	token, err = jwt.Parse(authToken, func(token *jwt.Token) (parsedSecret interface{}, parseErr error) {
		// Don't forget to validate the alg is what you expect:
		var ok bool
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			parseErr = fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			return
		}

		var claims jwt.MapClaims
		if claims, ok = token.Claims.(jwt.MapClaims); !ok {
			parseErr = fmt.Errorf("claims error")
			return
		}

		akI, ok := claims["access_key"]
		if !ok {
			parseErr = fmt.Errorf("signature validate failed, please input access_key")
			return
		}

		akVal, ok := akI.(string)
		if !ok {
			parseErr = fmt.Errorf("signature validate failed, access_key value type is not correct")
			return
		}

		svcUser, parseErr = getIdentityByAccessKey(akVal)

		if parseErr != nil {
			return
		}

		if svcUser.Id == 0 {
			parseErr = fmt.Errorf("signature validate failed, user is not exist")
			return
		}

		parsedSecret = []byte(svcUser.Secret)

		return

	})

	if err != nil {
		return
	}

	if !token.Valid {
		err = errors.New("signature is not valid")
		return
	}

	valid = true
	return
}

func GenerateToken(secret, accessKey string) (string, error) {
	// Create a new token object, specifying signing method and the claims
	// you would like it to contain.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"access_key": accessKey,
		"ts":         time.Now().Unix(),
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString([]byte(secret))
	return tokenString, err
}
