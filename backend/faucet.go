package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dpapathanasiou/go-recaptcha"
	"github.com/joho/godotenv"
	"github.com/tomasen/realip"
	"gitlab.com/faucet/src/types"
)

var chain string
var recaptchaSecretKey string
var amountFaucet string
var amountSteak string
var key string
var pass string
var node string
var publicUrl string

type claim_struct struct {
	Address  string `json:"address"`
	Response string `json:"response"`
}

func getEnv(key string) string {
	if value, ok := os.LookupEnv(key); ok {
		fmt.Println(key, "=", value)
		return value
	} else {
		log.Fatal("Error loading environment variable: ", key)
		return ""
	}
}

func main() {
	err := godotenv.Load(".env.local", ".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	chain = getEnv("FAUCET_CHAIN")
	recaptchaSecretKey = getEnv("FAUCET_RECAPTCHA_SECRET_KEY")
	amountFaucet = getEnv("FAUCET_AMOUNT_FAUCET")
	amountSteak = getEnv("FAUCET_AMOUNT_STEAK")
	key = getEnv("FAUCET_KEY")
	pass = getEnv("FAUCET_PASS")
	node = getEnv("FAUCET_NODE")
	publicUrl = getEnv("FAUCET_PUBLIC_URL")

	recaptcha.Init(recaptchaSecretKey)

	http.HandleFunc("/claim", getCoinsHandler)

	if err := http.ListenAndServe(publicUrl, nil); err != nil {
		log.Fatal("failed to start server", err)
	}
}

func executeCmd(command string, writes ...string) {
	cmd, wc, _ := goExecute(command)

	for _, write := range writes {
		wc.Write([]byte(write + "\n"))
	}
	cmd.Wait()
}

func goExecute(command string) (cmd *exec.Cmd, pipeIn io.WriteCloser, pipeOut io.ReadCloser) {
	cmd = getCmd(command)
	pipeIn, _ = cmd.StdinPipe()
	pipeOut, _ = cmd.StdoutPipe()
	go cmd.Start()
	time.Sleep(time.Second)
	return cmd, pipeIn, pipeOut
}

func getCmd(command string) *exec.Cmd {
	// split command into command and args
	split := strings.Split(command, " ")

	var cmd *exec.Cmd
	if len(split) == 1 {
		cmd = exec.Command(split[0])
	} else {
		cmd = exec.Command(split[0], split[1:]...)
	}

	return cmd
}

func getCoinsHandler(res http.ResponseWriter, request *http.Request) {
	address := request.FormValue("address")
	captchaResponse := request.FormValue("response")

	fmt.Println("No error", address, captchaResponse)

	(res).Header().Set("Access-Control-Allow-Origin", "*")

	// // make sure address is bech32
	// readableAddress, decodedAddress, decodeErr := bech32.DecodeAndConvert(claim.Address)
	// if decodeErr != nil {
	// 	panic(decodeErr)
	// }
	// // re-encode the address in bech32
	// encodedAddress, encodeErr := bech32.ConvertAndEncode(readableAddress, decodedAddress)
	// if encodeErr != nil {
	// 	panic(encodeErr)
	// }

	if len(address) != 40 {
		panic("Invalid address")
	}

	// make sure captcha is valid
	clientIP := realip.FromRequest(request)
	captchaPassed, captchaErr := recaptcha.Confirm(clientIP, captchaResponse)
	if captchaErr != nil {
		panic(captchaErr)
	}

	fmt.Println("Captcha passed? ", captchaPassed)

	// send the coins!
	if captchaPassed {
		sendFaucet := fmt.Sprintf(
			"akash send %v %v -k %v",
			amountFaucet, address, key)
		fmt.Println(time.Now().UTC().Format(time.RFC3339), address, "[1]")
		executeCmd(sendFaucet, pass)
	}

	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(types.SuccessResponse{
		Status: true,
		Data:   address,
	})

	return
}
