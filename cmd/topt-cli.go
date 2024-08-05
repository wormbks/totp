package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/atotto/clipboard"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"bksworm/totpcli/totpdb"
)

const (
	FLAG_SALT      = "salt"
	FLAG_ACCOUNT   = "account"
	FLAG_ISSUER    = "issuer"
	FLAG_URL       = "url"
	FLAG_IMAGE     = "image"
	FLAG_QRC       = "qrc"
	FLAG_DB        = "db"
	FLAG_CLIP      = "clipboard"
	FLAG_QUIET     = "quiet"
	PWD_PROMT      = "Enter password: "
	PWD_ERROR_WRAP = "error reading password: %w"
)

// githash is the Git commit hash of the current build.
// It is set during compilation using -ldflags.
var githash = "NONE"

// ReadPassword reads a password from the terminal without echoing it.
func ReadPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	fmt.Println() // Print a newline after the password input
	return string(bytePassword), nil
}

// conditionalPrintf prints a formatted string if the quiet flag is false.
func conditionalPrintf(quiet bool, format string, a ...interface{}) {
	if !quiet {
		fmt.Printf(format, a...)
	}
}

// GetSalt retrieves the salt from the command-line flag, environment variable, or default value.
func GetSalt(cmd *cobra.Command) []byte {
	const defaultSalt = "/* Copyright (c) 2024, Books Worm Limited. */"
	salt, _ := cmd.Flags().GetString(FLAG_SALT)
	if salt == "" {
		salt = viper.GetString(FLAG_SALT)
	}
	if salt == "" {
		salt = defaultSalt
	}
	return []byte(salt)
}

// getPwdSalt reads the password from the terminal and retrieves the salt value.
// It returns the password string, the salt as a byte slice, and any error that occurred.
func getPwdSalt(cmd *cobra.Command) (string, []byte, error) {
	pwd, err := ReadPassword(PWD_PROMT)
	if err != nil {
		return "", nil, fmt.Errorf(PWD_ERROR_WRAP, err)
	}
	return pwd, GetSalt(cmd), nil
}

// getQuiet returns the value of the "quiet" flag from the provided command.
// If the "quiet" flag is set, this function will return true, indicating that
// the program should run in a quiet mode and suppress non-essential output.
func getQuiet(cmd *cobra.Command) bool {
	val, _ := cmd.Flags().GetBool(FLAG_QUIET)
	return val
}

// getDBFilePath returns the database file path, checking command-line flag, environment variable, or default path
func getDBFilePath(cmd *cobra.Command) string {
	const defaultPath = "~/.config/totp-cli/entries.db"

	// Set up Viper to read environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("TOTP")
	viper.BindEnv("DB_PATH")

	quiet := getQuiet(cmd)

	// Check if the path is provided as a command-line argument
	var dbPath string
	if cmd.Flag(FLAG_DB).Changed {
		dbPath, _ = cmd.Flags().GetString(FLAG_DB)
	} else {
		// Check environment variable
		dbPath = viper.GetString("DB_PATH")
		if dbPath == "" {
			// Use the default path
			dbPath = defaultPath
		}
	}

	// Expand the `~` to the user's home directory
	if dbPath[:2] == "~/" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Println("Error getting user home directory:", err)
			os.Exit(1)
		}
		dbPath = filepath.Join(homeDir, dbPath[2:])
	}

	// Create the directory structure if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Println("Error creating directory structure:", err)
		os.Exit(1)
	}
	conditionalPrintf(quiet, "Using database file: %s\n", dbPath)

	return dbPath
}

var cmdCreateDb = &cobra.Command{
	Use:     "create-db",
	Aliases: []string{"c", "db"},
	Short:   "Create a new TOTP database",
	Long:    `Create a new TOTP database at the specified by flag "db" or environment variable "OTP_DB_PATH" path.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the database file path
		dbPath := getDBFilePath(cmd)
		// Initialize an empty TOTPData
		data := &totpdb.TOTPData{
			Entries: []totpdb.TOTPEntry{},
		}
		// Check if the database file already exists
		if _, err := os.Stat(dbPath); err == nil {
			return fmt.Errorf("database file already exists: %s", dbPath)
		}
		// Get the password and salt
		pwd, salt, err := getPwdSalt(cmd)
		if err != nil {
			return err
		}
		// Write the empty database to the specified path
		err = totpdb.WriteCBORSec(dbPath, data, pwd, salt)
		if err != nil {
			return fmt.Errorf("error creating database: %w", err)
		}
		quiet := getQuiet(cmd)
		conditionalPrintf(quiet, "Created new TOTP database at %s\n", dbPath)
		return nil
	},
}

var cmdAddUrl = &cobra.Command{
	Use:     "add-url",
	Aliases: []string{"a"},
	Short:   "Add a new TOTP",
	Long:    `Add a new TOTP from a command-line URL or clipboard.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		url, _ := cmd.Flags().GetString(FLAG_URL)
		clipboardInput, _ := cmd.Flags().GetBool(FLAG_CLIP)

		if url == "" && !clipboardInput {
			// Read from clipboard if URL is not provided
			url, err = clipboard.ReadAll()
			if err != nil {
				return fmt.Errorf("error reading from clipboard: %w", err)
			}
		}

		key, err := otp.NewKeyFromURL(url)
		if err != nil {
			return fmt.Errorf("error parsing TOPT URL: %w", err)
		}

		dbFilePath := getDBFilePath(cmd)
		// Get the password and salt
		pwd, salt, err := getPwdSalt(cmd)
		if err != nil {
			return err
		}
		data, err := totpdb.ReadCBORSec(dbFilePath, pwd, salt)
		if err != nil {
			return fmt.Errorf("error reading TOTP data: %w", err)
		}

		if err := data.AddEntry(key); err != nil {
			return fmt.Errorf("error adding for %s from %s: %w", key.AccountName(), key.Issuer(), err)
		}
		// Write the empty database to the specified path
		if err := totpdb.WriteCBORSec(dbFilePath, data, pwd, salt); err != nil {
			return fmt.Errorf("error writing TOTP data: %w", err)
		}

		quiet := getQuiet(cmd)
		conditionalPrintf(quiet, "Added TOTP for %s from %s\n", key.AccountName(), key.Issuer())
		// Generate the TOTP code
		code, err := totp.GenerateCode(key.Secret(), time.Now())
		if err != nil {
			return fmt.Errorf("error generating TOTP code: %w", err)
		}

		// Print the TOTP code
		conditionalPrintf(quiet, "Generated TOTP code: ")
		fmt.Println(code)

		return nil

	},
}

var cmdList = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l"},
	Short:   "List all TOTPs",
	Long:    `List all TOTPs in the database as an ASCII table.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath := getDBFilePath(cmd)
		// Get the password and salt
		pwd, salt, err := getPwdSalt(cmd)
		if err != nil {
			return err
		}

		data, err := totpdb.ReadCBORSec(dbPath, pwd, salt)
		if err != nil {
			return fmt.Errorf("error reading TOTP data: %w", err)
		}
		data.PrintTable()

		return nil
	},
}

var cmdGenerate = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"gen", "g"},
	Short:   "Generate a TOTP",
	Long:    `Generate a TOTP for the specified account and issuer.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		account, _ := cmd.Flags().GetString(FLAG_ACCOUNT)
		issuer, _ := cmd.Flags().GetString(FLAG_ISSUER)
		publish, _ := cmd.Flags().GetBool(FLAG_CLIP)

		dbFilePath := getDBFilePath(cmd)
		// Get the password and salt
		pwd, salt, err := getPwdSalt(cmd)
		if err != nil {
			return err
		}
		data, err := totpdb.ReadCBORSec(dbFilePath, pwd, salt)
		if err != nil {
			return fmt.Errorf("error reading TOTP data: %w", err)
		}

		val, err := data.GetEntry(account, issuer)
		if err != nil {
			return fmt.Errorf("account not found: %w", err)
		}

		code, err := totp.GenerateCode(val.Secret, time.Now())
		if err != nil {
			return fmt.Errorf("error generating TOTP: %w", err)
		}

		// Print the TOTP code
		quiet := getQuiet(cmd)
		if quiet {
			fmt.Println(code)
		} else {
			conditionalPrintf(quiet, "TOTP for %s from %s: %s\n",
				val.AccountName, val.Issuer, code)
		}
		if publish {
			err = clipboard.WriteAll(code)
			if err != nil {
				return fmt.Errorf("error writing to clipboard: %w", err)
			}
			conditionalPrintf(quiet, "Copied TOTP code to clipboard\n")
		}

		return nil
	},
}

var cmdRremove = &cobra.Command{
	Use:     "remove",
	Aliases: []string{"rm"},
	Short:   "Remove a TOTP by account and issuer",
	Long:    `Remove a TOTP by account and issuer.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		account, _ := cmd.Flags().GetString(FLAG_ACCOUNT)
		issuer, _ := cmd.Flags().GetString("issuer")

		dbFilePath := getDBFilePath(cmd)
		// Get the password and salt
		pwd, salt, err := getPwdSalt(cmd)
		if err != nil {
			return err
		}

		data, err := totpdb.ReadCBORSec(dbFilePath, pwd, salt)
		if err != nil {
			return fmt.Errorf("error reading TOTP data: %w", err)
		}

		if err := data.RemoveEntry(account, issuer); err != nil {
			return fmt.Errorf("error removing TOTP: %w", err)
		}

		if err := totpdb.WriteCBORSec(dbFilePath, data, pwd, salt); err != nil {
			return fmt.Errorf("error writing TOTP data: %w", err)
		}

		quiet := getQuiet(cmd)
		conditionalPrintf(quiet, "Removed TOTP for %s from %s\n", account, issuer)

		return nil
	},
}

var cmdAddQRC = &cobra.Command{
	Use:     "add-qrc",
	Aliases: []string{"qrc"},
	Short:   "Add a new TOTP as QR Code",
	Long:    `Add a new TOTP as QR Code from file specified by flag "image".`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fileName, _ := cmd.Flags().GetString(FLAG_IMAGE)

		file, err := os.Open(fileName)
		if err != nil {
			return fmt.Errorf("error opening image file: %w", err)
		}
		defer file.Close()

		// Decode the image to extract the QR code data
		img, _, err := image.Decode(file)
		if err != nil {
			return fmt.Errorf("error decoding image: %w", err)
		}
		// prepare BinaryBitmap
		bmp, err := gozxing.NewBinaryBitmapFromImage(img)
		if err != nil {
			return fmt.Errorf("error creating BinaryBitmap: %w", err)
		}

		// decode image
		qrReader := qrcode.NewQRCodeReader()
		result, err := qrReader.Decode(bmp, nil)
		if err != nil {
			return fmt.Errorf("error decoding QR code: %w", err)
		}

		// Extract the TOTP URL from the QR code data
		totpURL := result.GetText()

		// Parse the TOTP URL to extract account name, issuer, and secret
		key, err := otp.NewKeyFromURL(totpURL)
		if err != nil {
			return fmt.Errorf("error parsing TOTP URL: %w", err)
		}

		// Generate the TOTP code
		code, err := totp.GenerateCode(key.Secret(), time.Now())
		if err != nil {
			return fmt.Errorf("error generating TOTP code: %w", err)
		}

		quiet := getQuiet(cmd)
		// Print the TOTP code
		conditionalPrintf(quiet, "Generated TOTP code: ")
		fmt.Println(code)

		dbFilePath := getDBFilePath(cmd)
		// Get the password and salt
		pwd, salt, err := getPwdSalt(cmd)
		if err != nil {
			return err
		}

		// Add the TOTP to the database
		data, err := totpdb.ReadCBORSec(dbFilePath, pwd, salt)
		if err != nil {
			return fmt.Errorf("error reading TOTP data: %w", err)
		}
		if err := data.AddEntry(key); err != nil {
			return fmt.Errorf("error adding for %s from %s: %w", key.AccountName(), key.Issuer(), err)
		}

		if err := totpdb.WriteCBORSec(dbFilePath, data, pwd, salt); err != nil {
			return fmt.Errorf("error writing TOTP data: %w", err)
		}

		conditionalPrintf(quiet, "Added TOTP for %s from %s\n", key.AccountName(), key.Issuer())

		return nil
	},
}

var rootCmd = &cobra.Command{
	Use:   "totp",
	Short: "TOTP CLI app",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		quiet := getQuiet(cmd)
		cmd.SilenceUsage = quiet
		cmd.SilenceErrors = quiet
	},
}

func setCobraCommands() {
	rootCmd.AddCommand(cmdAddUrl, cmdList, cmdGenerate, cmdRremove, cmdAddQRC, cmdCreateDb)

	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress output")
	viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet"))

	rootCmd.PersistentFlags().StringP(FLAG_DB, "d", "", "Path to the database file, if not set in, environment variable TOTP_DB_PRTH or defaulting to ~/.config/totp-cli/entries.db")
	viper.BindPFlag(FLAG_DB, rootCmd.PersistentFlags().Lookup(FLAG_DB))
	rootCmd.PersistentFlags().StringP(FLAG_SALT, "s", "", "Salt input for encryption or, if not set in, environment variable TOTP_SALT or default value")
	viper.BindPFlag(FLAG_SALT, rootCmd.PersistentFlags().Lookup(FLAG_SALT))
	viper.SetDefault(FLAG_SALT, os.Getenv("TOTP_SALT"))

	cmdAddUrl.Flags().StringP(FLAG_URL, "u", "", "OTP URL to add. It must be in \"\".")
	cmdAddUrl.Flags().BoolP(FLAG_CLIP, "c", false, "Read OTP URL from clipboard")

	cmdAddQRC.Flags().StringP(FLAG_IMAGE, "i", "", "Read OTP image from file")
	cmdAddQRC.MarkFlagRequired(FLAG_IMAGE)

	cmdGenerate.Flags().StringP(FLAG_ACCOUNT, "a", "", "Account name to generate TOTP for")
	cmdGenerate.Flags().StringP(FLAG_ISSUER, "i", "", "Issuer name to generate TOTP for")
	cmdGenerate.Flags().BoolP(FLAG_CLIP, "c", false, "Put code to clipboard")
	cmdGenerate.MarkFlagRequired(FLAG_ACCOUNT)

	cmdRremove.Flags().StringP(FLAG_ACCOUNT, "a", "", "Account name to remove TOTP for")
	cmdRremove.Flags().StringP(FLAG_ISSUER, "i", "", "Issuer name to remove TOTP for")
	cmdRremove.MarkFlagRequired(FLAG_ACCOUNT)

}

func main() {
	setCobraCommands()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
