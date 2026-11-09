// quetoo-rcon is a command-line remote-console client for Quetoo game servers.
// The server argument is an alias to a server record in the text-format
// proto config file. The default location for this file is ~/.quetoo.pb
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/packetflinger/libquetoo/state"
	"google.golang.org/protobuf/encoding/prototext"

	pb "github.com/packetflinger/libquetoo/proto"
)

// Temp server structure. All info needed for sending rcon msgs
type Server struct {
	Name     string
	Addr     string
	Password string
}

var (
	serversFile = ".quetoo.pb" // default name, should be in home directory
	config      = flag.String("config", "", "Specify a server data file")
)

func main() {
	flag.Parse()
	if len(flag.Args()) < 2 {
		showUsage()
		return
	}

	serverspb, err := loadConfig()
	if err != nil {
		log.Println(err)
		return
	}

	pw, addr, port, err := resolveTarget(serverspb, flag.Arg(0))
	if err != nil {
		log.Println(err)
		return
	}

	server := state.Server{
		Address:  addr,
		Port:     port,
		Password: pw,
	}

	command := strings.Join(flag.Args()[1:], " ")

	rcon, err := server.DoRcon(command)
	if err != nil {
		log.Println(err)
		return
	}

	if len(rcon.Output) > 0 {
		if isTerminal(os.Stdout) {
			fmt.Println(colorize(rcon.Output))
		} else {
			fmt.Println(stripColor(rcon.Output))
		}
	}
}

// Read the text-format proto config file and unmarshal it
func loadConfig() (*pb.ServerFile, error) {
	cfg := &pb.ServerFile{}

	if *config == "" {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		*config = path.Join(homedir, serversFile)
	}

	raw, err := os.ReadFile(*config)
	if err != nil {
		return nil, err
	}

	err = prototext.Unmarshal(raw, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// Attempt to match the target arg to an identifier in the server config. This
// should be a case-insensitive match.
//
// Returns the password, ip/host, port, and any errors
func resolveTarget(cfg *pb.ServerFile, targ string) (string, string, int, error) {
	for _, sv := range cfg.GetServer() {
		if strings.EqualFold(sv.GetName(), targ) {
			for _, pw := range cfg.GetPassword() {
				if pw.Name == sv.GetRconPassword() {
					tokens := strings.Split(sv.GetAddress(), ":")
					if len(tokens) != 2 {
						return "", "", 0, errors.New("invalid address for server - " + sv.GetAddress())
					}
					port, err := strconv.Atoi(tokens[1])
					if err != nil {
						return "", "", 0, errors.New("invalid port - " + sv.GetAddress())
					}
					return pw.Secret, tokens[0], port, nil
				}
			}
		}
	}

	return "", "", 0, errors.New("can't resolve alias - " + targ)
}

func showUsage() {
	fmt.Printf("Usage: %s [flags] <server> <command>\n", os.Args[0])
	fmt.Println("flags:")
	flag.PrintDefaults()
	fmt.Println("server\n", "  A server alias from the servers text-proto file.")
	fmt.Println("command\n", "  The command to be executed on the remote server")
}

// Quake 3 style color codes (^0-^9) and their ANSI equivalents.
var (
	ansiColors = strings.NewReplacer(
		"^0", "\x1b[30m", // black
		"^1", "\x1b[31m", // red
		"^2", "\x1b[32m", // green
		"^3", "\x1b[33m", // yellow
		"^4", "\x1b[34m", // blue
		"^5", "\x1b[36m", // cyan
		"^6", "\x1b[35m", // magenta
		"^7", "\x1b[39m", // white (terminal default)
		"^8", "\x1b[38;5;208m", // orange
		"^9", "\x1b[90m", // grey
	)
	noColors = strings.NewReplacer(
		"^0", "", "^1", "", "^2", "", "^3", "", "^4", "",
		"^5", "", "^6", "", "^7", "", "^8", "", "^9", "",
	)
)

// colorize converts color codes to ANSI escapes, resetting at the end.
func colorize(in string) string {
	return ansiColors.Replace(in) + "\x1b[0m"
}

func stripColor(in string) string {
	return noColors.Replace(in)
}

// isTerminal reports whether f is a terminal, so piped output stays plain.
func isTerminal(f *os.File) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
