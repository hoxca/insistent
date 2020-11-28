package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"insistent/baresip"
	"io"
	"os/exec"
	"strings"
	"time"
)

var ncall = 0
var cmd *exec.Cmd
var pipe io.ReadCloser

func main() {

	numberPtr := flag.String("number", "0323455329", "Phone number to call")
	sipProxyPtr := flag.String("sipproxy", "21.17.54.38", "Sip Proxy IP")
	flag.Parse()
	dialQuery := fmt.Sprintf("d%s@%s", *numberPtr, *sipProxyPtr)

	baresip.Mock = true
	baresip.Path = "/usr/local/bin/baresip"
	baresip.Config = "/Users/hugh/.baresip/"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd, pipe = baresip.Launch(ctx)
	scanResult(cmd, pipe)

	time.Sleep(1 * time.Second)
	baresip.Call(ncall, dialQuery)

	// should not be there !
	baresip.Close(cmd)
}

func scanResult(cmd *exec.Cmd, pipe io.ReadCloser) {
	scanner := bufio.NewScanner(pipe)
	go func() {
		fmt.Println("Scan goroutine")
		for scanner.Scan() {
			if checkResult(scanner.Text()) {
				break
			}
		}
		baresip.Close(cmd)
	}()
}

func checkResult(result string) bool {

	fmt.Printf("%s\n", result)
	if strings.Contains(result, "terminated") {
		fmt.Println("Call terminated !")
		return false
	}

	if strings.Contains(result, "Call established") {
		fmt.Println("Ok, Call answered!")
		return true
	}

	if strings.Contains(result, "Call in-progress") {
		fmt.Println("Call in progress")
	}
	return false
}
