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
		fmt.Println(stripColor(rcon.Output))
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

func stripColor(in string) string {
	out := strings.ReplaceAll(in, "^1", "")
	out = strings.ReplaceAll(out, "^2", "")
	out = strings.ReplaceAll(out, "^3", "")
	out = strings.ReplaceAll(out, "^4", "")
	out = strings.ReplaceAll(out, "^5", "")
	out = strings.ReplaceAll(out, "^6", "")
	out = strings.ReplaceAll(out, "^7", "")
	out = strings.ReplaceAll(out, "^8", "")
	out = strings.ReplaceAll(out, "^9", "")
	return out
}
