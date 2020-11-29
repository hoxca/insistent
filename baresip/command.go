package baresip

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"

	Log "github.com/apatters/go-conlog"
)

// Mock activate the baresip mock
var Mock bool
var (
	// Path store the baresip exec path
	Path string
	// Config store the baresip default configuration
	Config string
)

// Hangup function stop the call
func Hangup() {

	if !Mock {
		baresipHangupURI := "http://127.0.0.1:8000/?b"
		resp, err := http.Get(baresipHangupURI)
		if err != nil {
			return
		}
		defer resp.Body.Close()
	}
	fmt.Println("Hangup the phone!")
}

// Close will stop the baresip process
func Close(cmd *exec.Cmd) {

	if !Mock {
		baresipQuitURI := "http://127.0.0.1:8000/?q"
		resp, err := http.Get(baresipQuitURI)
		if err != nil {
			return
		}
		defer resp.Body.Close()
	}

	fmt.Println("Closing baresip")
	cmd.Process.Kill()
	os.Exit(1)
}

// Call is a recursive function that make the call and repeat it, 3 times before hangup
func Call(nc int, callee string, dialQuery string) {

	fmt.Printf("Calling %s iteration:%d\n", callee, nc)
	baresipQuery := fmt.Sprintf("http://127.0.0.1:8000/?%s", url.QueryEscape(dialQuery))

	if !Mock {
		resp, err := http.Get(baresipQuery)
		if err != nil {
			return
		}
		defer resp.Body.Close()
	}

	waitForAnswer(nc)
	if nc == 2 {
		fmt.Println("Wake up failed 3 times :(")
		return
	}
	nc++
	Call(nc, callee, dialQuery)
}

// Launch function will start the baresip process
func Launch(ctx context.Context) (*exec.Cmd, io.ReadCloser) {

	Log.Debugf("baresip path: %s", Path)
	Log.Debugf("baresip config: %s", Config)

	var cmd *exec.Cmd
	if Mock {
		Log.Debugf("Mocking call...")
		cmd = exec.CommandContext(ctx, "/usr/bin/tail", "-500f", "data/data.txt")
	}
	if !Mock {
		cmd = exec.CommandContext(ctx, Path, "-f", Config)
	}

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("pipe error %v", err)
	}
	err = cmd.Start()
	if err != nil {
		fmt.Println("Fatal error")
		os.Exit(1)
	}

	time.Sleep(1 * time.Second)
	return cmd, pipe
}

func waitForAnswer(iteration int) {
	var count = 0
	for range time.Tick(time.Second) {
		count++
		if count == 15 {
			fmt.Printf("Call timeout... ")
			Hangup()
			time.Sleep(5 * time.Second)
			break
		}
	}
	return
}
