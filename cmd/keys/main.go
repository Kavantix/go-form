package main

import (
	"crypto/ed25519"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"

	"github.com/Kavantix/go-form/sign"
	"github.com/rsc/getopt"
)

func eprintln(line string) {
	fmt.Fprintln(os.Stderr, line)
}

func efatalln(line string) {
	fmt.Fprintln(os.Stderr, line)
	os.Exit(1)
}

func eprintf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
}

func efatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

var (
	name  = flag.String("name", "key", "The name of the key.")
	dir   = flag.String("directory", "", "The directory to store the key.")
	help  = flag.Bool("help", false, "Shows this help message.")
	force = flag.Bool("force", false, "Force will allow overwriting existing key files")
)

func failWithUsage() {
	eprintf("%s is a cli utility to handle signing keys\n", os.Args[0])
	eprintln("")
	eprintln("Sub commands:")
	eprintln("    generate")
	eprintln("        Generates a new key")
	eprintln("Flags:")
	getopt.PrintDefaults()
	os.Exit(1)
}

func setupArgs() {
	getopt.Alias("n", "name")
	getopt.Alias("d", "directory")
	getopt.Alias("h", "help")
	getopt.Alias("f", "force")
	getopt.Parse()

	if *help {
		failWithUsage()
	}
}

func main() {
	setupArgs()

	subcommand := flag.Arg(0)
	switch subcommand {
	case "generate":
		generateKey()
	case "jwt":
		privPath, pubPath := paths()
		err := sign.LoadKeys(privPath, pubPath)
		if err != nil {
			efatalf("Failed to load keys: %s\n", err.Error())
		}
		result, err := sign.CreateJwt(&sign.JwtOptions{
			Subject: "userid",
		})
		if err != nil {
			efatalf("Failed to create jwt: %s\n", err)
		}
		_, err = sign.ParseJwt(result)
		if err != nil {
			efatalf("Failed to validate jwt: %s\n", err.Error())
		}
		fmt.Println(result)

	default:
		eprintln("Missing subcommand")
		failWithUsage()
	}
}

func paths() (privPath, pubPath string) {

	path := "./"
	if dir != nil && *dir != "" {
		err := os.MkdirAll(*dir, 0700)
		if err != nil {
			efatalf("Failed to create directory `%s`: %s\n", *dir, err.Error())
		}
		path = *dir
		path += "/"
	}
	path += *name
	privPath = path + ".priv"
	pubPath = path + ".pub"
	return
}

func generateKey() {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		eprintf("Failed to generate key: %s\n", err.Error())
		os.Exit(1)
	}
	privPath, pubPath := paths()
	if !*force {
		_, privErr := os.Stat(privPath)
		_, pubErr := os.Stat(pubPath)
		if !errors.Is(privErr, fs.ErrNotExist) || !errors.Is(pubErr, fs.ErrNotExist) {
			eprintln("Key already exists, use --force to generate anyways.")
			os.Exit(1)
		}
	}
	os.WriteFile(privPath, sign.Base64Encode(privateKey), 0700)
	os.WriteFile(pubPath, sign.Base64Encode(publicKey), 0700)

}
