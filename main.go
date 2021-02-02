/*

midicat is a program that transfers MIDI data between midi ports and stdin/stdout.
The idea is, that you can have midi libraries that do not depend on c (or CGO in the case of go)
and still might want to use some midi to ports. But maybe it is just an option that is not
used much and we don't want to bother the other users with a c/CGO dependency.

example

midicat in -i=10 | midicat log | midicat out -i=11

(routes midi from midi in port 10 to midi out port 11 while logging the parsed messages in readable way to stderr)

*/

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"runtime"

	"gitlab.com/gomidi/midi"
	"gitlab.com/gomidi/midi/midimessage/realtime"
	"gitlab.com/gomidi/midi/midireader"
	"gitlab.com/gomidi/rtmididrv"
	"gitlab.com/metakeule/config"
)

var (
	cfg = config.MustNew("midicat", VERSION, "midicat transfers MIDI data between midi ports and stdin/stdout")

	argPortNum  = cfg.NewInt32("index", "index of the midi port. Only specify either the index or the name. If neither is given, the first port is used.", config.Shortflag('i'))
	argPortName = cfg.NewString("name", "name of the midi port. Only specify either the index or the name. If neither is given, the first port is used.")
	argJson     = cfg.NewBool("json", "return the list in JSON format")

	cmdIn  = cfg.MustCommand("in", "read midi from an in port and print it to stdout").Skip("json")
	cmdOut = cfg.MustCommand("out", "read midi from stdin and print it to an out port").Skip("json")

	cmdIns  = cfg.MustCommand("ins", "show the available midi in ports").SkipAllBut("json")
	cmdOuts = cfg.MustCommand("outs", "show the available midi out ports").SkipAllBut("json")

	cmdLog = cfg.MustCommand("log", "pass the midi from stdin to stdout while logging it to stderr").SkipAllBut()

	shouldStop = make(chan bool, 1)
	didStop    = make(chan bool, 1)
)

func main() {
	err := run()

	if err != nil {
		fmt.Fprintln(os.Stderr, cfg.Usage())
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err.Error())
		os.Exit(1)
	}

	os.Exit(0)
}

func run() error {
	runtime.GOMAXPROCS(1)
	err := cfg.Run()

	if err != nil {
		return err
	}

	if cfg.ActiveCommand() == cmdLog {
		return log()
	}

	drv, err := rtmididrv.New()
	if err != nil {
		return err
	}

	switch cfg.ActiveCommand() {
	case cmdIns:
		if argJson.Get() {
			return showInJson(drv)
		} else {
			return showInPorts(drv)
		}
	case cmdOuts:
		if argJson.Get() {
			return showOutJson(drv)
		} else {
			return showOutPorts(drv)
		}
	case cmdIn:
		return runIn(drv)
	case cmdOut:
		return runOut(drv)
	default:
		return fmt.Errorf("[command] missing")
	}
}

func logRealTime(rt realtime.Message) {
	fmt.Fprintf(os.Stderr, "%s\n", rt)
}

func log() error {
	for {
		var b = make([]byte, 3)
		_, err := os.Stdin.Read(b)
		if err != nil {
			break
		}
		_, werr := os.Stdout.Write(b)
		if werr != nil {
			fmt.Fprintf(os.Stderr, "could not pass % X from stdin to stdout: %s\n", b, werr.Error())
		}

		msg, merr := midireader.New(bytes.NewReader(b), logRealTime).Read()

		if merr != nil {
			fmt.Fprintf(os.Stderr, "could not parse % X from stdin: %s\n", b, merr.Error())
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", msg)
		}
	}
	return nil
}

func runIn(drv midi.Driver) error {
	defer drv.Close()
	in, err := midi.OpenIn(drv, int(argPortNum.Get()), argPortName.Get())

	if err != nil {
		return err
	}

	err = in.SetListener(func(data []byte, deltaMicroseconds int64) {
		_, werr := os.Stdout.Write(data)
		_ = werr
	})

	if err != nil {
		return err
	}

	sigchan := make(chan os.Signal, 10)

	// listen for ctrl+c
	go signal.Notify(sigchan, os.Interrupt)

	// interrupt has happend
	<-sigchan
	in.StopListening()
	return nil
}

func runOut(drv midi.Driver) error {
	defer drv.Close()
	out, err := midi.OpenOut(drv, int(argPortNum.Get()), argPortName.Get())

	if err != nil {
		return err
	}

	for {
		var b = make([]byte, 3)
		_, err = os.Stdin.Read(b)
		if err != nil {
			break
		}
		_, werr := out.Write(b)
		if werr != nil {
			fmt.Fprintf(os.Stderr, "could not write % X to port %q: %s\n", b, out.String(), werr.Error())
		}
	}

	return nil
}

func showInJson(drv midi.Driver) error {
	defer drv.Close()
	ports, err := drv.Ins()

	if err != nil {
		return err
	}

	var portm = map[int]string{}

	for _, port := range ports {
		portm[port.Number()] = port.String()
	}

	enc := json.NewEncoder(os.Stdout)
	return enc.Encode(portm)
}

func showInPorts(drv midi.Driver) error {
	defer drv.Close()
	ins, err := drv.Ins()

	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, "MIDI inputs")

	for _, in := range ins {
		fmt.Fprintf(os.Stdout, "[%v] %s\n", in.Number(), in.String())
	}

	return nil
}

func showOutJson(drv midi.Driver) error {
	defer drv.Close()
	ports, err := drv.Outs()

	if err != nil {
		return err
	}

	var portm = map[int]string{}

	for _, port := range ports {
		portm[port.Number()] = port.String()
	}

	enc := json.NewEncoder(os.Stdout)
	return enc.Encode(portm)
}

func showOutPorts(drv midi.Driver) error {
	defer drv.Close()
	outs, err := drv.Outs()

	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, "MIDI outputs")

	for _, out := range outs {
		fmt.Fprintf(os.Stdout, "[%v] %s\n", out.Number(), out.String())
	}

	return nil
}
