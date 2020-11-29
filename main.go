package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"insistent/baresip"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	Log "github.com/apatters/go-conlog"
	"github.com/spf13/viper"
)

var (
	cfgFileNotFound = false
	ncall           = 0
	cmd             *exec.Cmd
	pipe            io.ReadCloser
	verbosityPtr    *string
)

type stringFlag struct {
	set   bool
	value string
}

func (sf *stringFlag) Set(x string) error {
	sf.value = x
	sf.set = true
	return nil
}

func (sf *stringFlag) String() string {
	return sf.value
}

func (sf *stringFlag) Split() []string {
	return strings.Split(sf.value, ",")
}

var callees stringFlag

func main() {

	flag.Var(&callees, "callees", "comma separated list of callee by order of call")
	verbosityPtr = flag.String("verbosity", "warn", "Log level (debug, info, warn, error)")
	flag.Parse()

	setUpLogs()
	viper := readConfig()

	sipProxy := viper.GetString("sipProxy")
	numbers := viper.GetStringMap("callees")
	baresip.Path = viper.GetString("baresip.path")
	baresip.Config = viper.GetString("baresip.config")
	baresip.Mock = true

	callees := sanityCheck(callees, numbers)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd, pipe = baresip.Launch(ctx)
	scanResult(cmd, pipe)

	time.Sleep(1 * time.Second)

	for c := range callees {
		dialQuery := fmt.Sprintf("d%s@%s", numbers[callees[c]], sipProxy)
		baresip.Call(ncall, callees[c], dialQuery)
	}

	// should not be there !
	baresip.Close(cmd)
}

func sanityCheck(calleesFlag stringFlag, numbers map[string]interface{}) []string {
	if len(calleesFlag.String()) == 0 {
		flag.Usage()
		fmt.Println("\nError: Please provide one or more callee name(s)")
	}
	callees := calleesFlag.Split()
	var (
		wrongMapping = false
		errMsg       string
	)
	for c := range callees {
		if numbers[callees[c]] == nil {
			wrongMapping = true
			errMsg = fmt.Sprintf("%s\nError: You must provide number mapping in configuration for callee: %s", errMsg, callees[c])
		}
		Log.Debugf("Call %s at number: %s", callees[c], numbers[callees[c]])
	}
	if wrongMapping {
		flag.Usage()
		fmt.Printf("%s", errMsg)
	}
	return callees
}

func readConfig() *viper.Viper {
	v := viper.New()
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	confdir := fmt.Sprintf("%s/conf", dir)
	// if we came from bin directory
	confdir1 := fmt.Sprintf("%s/../conf", dir)
	// Search yaml config file in program path with name "insistent.yaml".
	v.AddConfigPath(confdir)
	v.AddConfigPath(confdir1)
	v.SetConfigType("yaml")
	v.SetConfigName("insistent")
	//	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			cfgFileNotFound = true
			fmt.Println("Config file not found")
		} else {
			Log.Debug("Something look strange")
			Log.Debugf("error: %v\n", err)
		}
	} else {
		Log.Debugf("Using config file: %s", v.ConfigFileUsed())
	}
	return v
}

func scanResult(cmd *exec.Cmd, pipe io.ReadCloser) {
	scanner := bufio.NewScanner(pipe)
	go func() {
		Log.Debugf("Scan goroutine")
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

func setUpLogs() {
	formatter := Log.NewStdFormatter()
	formatter.Options.LogLevelFmt = Log.LogLevelFormatLongTitle
	Log.SetFormatter(formatter)
	switch *verbosityPtr {
	case "debug":
		Log.SetLevel(Log.DebugLevel)
	case "info":
		Log.SetLevel(Log.InfoLevel)
	case "warn":
		Log.SetLevel(Log.WarnLevel)
	case "error":
		Log.SetLevel(Log.ErrorLevel)
	default:
		Log.SetLevel(Log.WarnLevel)
	}
}
