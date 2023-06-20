package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vulcanize/lotus-index-attestation/pkg/attestation"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
)

var (
	cfgFile        string
	envFile        string
	subCommand     string
	logWithCommand log.Entry
)

var rootCmd = &cobra.Command{
	Use:              "lotus-index-attestation",
	PersistentPreRun: initFuncs,
}

func Execute() {
	log.Info("----- Starting Lotus index attestation service -----")
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func initFuncs(cmd *cobra.Command, args []string) {
	logInit()

	// TODO: add metrics
	/*
		if viper.GetBool("metrics") {
			prom.Init()
		}

		if viper.GetBool("prom.http") {
			addr := fmt.Sprintf(
				"%s:%s",
				viper.GetString("prom.http.addr"),
				viper.GetString("prom.http.port"),
			)
			prom.Serve(addr)
		}
	*/
}

func init() {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file location")
	rootCmd.PersistentFlags().StringVar(&envFile, "env", "", "environment file location")

	rootCmd.PersistentFlags().String("log-level", log.InfoLevel.String(), "log level (trace, debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().String("log-file", "", "file path for logging")

	rootCmd.PersistentFlags().String("checksum-db-directory", "", "path for directory that contains a checksums.db")
	rootCmd.PersistentFlags().Uint("checksum-chunk-size", 2880, "epoch range size for caluclating checksums over")
	rootCmd.PersistentFlags().Bool("checksum-on", true, "turn checksumming on")

	rootCmd.PersistentFlags().Bool("server-on", false, "turn on the http rpc server")
	rootCmd.PersistentFlags().String("server-port", "8087", "port http rpc server")

	viper.BindPFlag(attestation.LOG_LEVEL_TOML, rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag(attestation.LOG_FILE_TOML, rootCmd.PersistentFlags().Lookup("log-file"))

	viper.BindPFlag(attestation.CHECKSUM_DB_DIRECTORY_TOML, rootCmd.PersistentFlags().Lookup("checksum-db-directory"))
	viper.BindPFlag(attestation.CHECKSUM_CHUNK_SIZE_TOML, rootCmd.PersistentFlags().Lookup("checksum-chunk-size"))
	viper.BindPFlag(attestation.SUPPORTS_CHECKSUMMING_TOML, rootCmd.PersistentFlags().Lookup("checksum-on"))

	viper.BindPFlag(attestation.SERVER_PORT_TOML, rootCmd.PersistentFlags().Lookup("server-port"))
	viper.BindPFlag(attestation.SUPPORTS_SERVER_TOML, rootCmd.PersistentFlags().Lookup("server-on"))

	// TODO: add metrics
	/*
		rootCmd.PersistentFlags().Bool("metrics", false, "enable metrics")

		rootCmd.PersistentFlags().Bool("prom-http", false, "enable http service for prometheus")
		rootCmd.PersistentFlags().String("prom-http-addr", "127.0.0.1", "http host for prometheus")
		rootCmd.PersistentFlags().String("prom-http-port", "8090", "http port for prometheus")

		viper.BindPFlag("metrics", rootCmd.PersistentFlags().Lookup("metrics"))

		viper.BindPFlag("prom.http", rootCmd.PersistentFlags().Lookup("prom-http"))
		viper.BindPFlag("prom.http.addr", rootCmd.PersistentFlags().Lookup("prom-http-addr"))
		viper.BindPFlag("prom.http.port", rootCmd.PersistentFlags().Lookup("prom-http-port"))
	*/
}

func loadConfig() {
	if cfgFile == "" && envFile == "" {
		log.Fatal("No configuration file specified, use --config , --env flag to provide configuration")
	}

	if cfgFile != "" {
		if filepath.Ext(cfgFile) != ".toml" {
			log.Fatal("Provide .toml file for --config flag")
		}

		viper.SetConfigFile(cfgFile)
		if err := viper.ReadInConfig(); err != nil {
			log.Fatalf("Couldn't read config file: %s", err.Error())
		}

		log.Infof("Using config file: %s", viper.ConfigFileUsed())
	}

	if envFile != "" {
		if filepath.Ext(envFile) != ".env" {
			log.Fatal("Provide .env file for --env flag")
		}

		if err := godotenv.Load(envFile); err != nil {
			log.Fatalf("Failed to set environment variable from env file: %s", err.Error())
		}

		log.Infof("Using env file: %s", envFile)
	}
}

func logInit() error {
	// Set the output.
	viper.BindEnv(attestation.LOG_FILE_TOML, attestation.LOG_FILE)
	logFile := viper.GetString("log.file")
	if logFile != "" {
		file, err := os.OpenFile(logFile,
			os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
		if err == nil {
			log.Infof("Directing output to %s", logFile)
			log.SetOutput(file)
		} else {
			log.SetOutput(os.Stdout)
			log.Info("Failed to logrus.to file, using default stdout")
		}
	} else {
		log.SetOutput(os.Stdout)
	}

	// Set the level.
	viper.BindEnv(attestation.LOG_LEVEL_TOML, attestation.LOG_LEVEL)
	lvl, err := log.ParseLevel(viper.GetString("log.level"))
	if err != nil {
		return err
	}
	log.SetLevel(lvl)

	formatter := &log.TextFormatter{
		FullTimestamp: true,
	}
	// Show file/line number only at Trace level.
	if lvl >= log.TraceLevel {
		log.SetReportCaller(true)

		// We need to exclude this wrapper code, logrus.us itself, and the runtime from the stack to show anything useful.
		// cf. https://github.com/sirupsen/logrus.us/pull/973
		formatter.CallerPrettyfier = func(frame *runtime.Frame) (function string, file string) {
			pcs := make([]uintptr, 50)
			_ = runtime.Callers(0, pcs)
			frames := runtime.CallersFrames(pcs)

			// Filter logrus.wrapper / logrus.us / runtime frames.
			for next, again := frames.Next(); again; next, again = frames.Next() {
				if !strings.Contains(next.File, "sirupsen/logrus.us") &&
					!strings.HasPrefix(next.Function, "runtime.") &&
					!strings.Contains(next.File, "lotus-index-attestation/pkg/log") {
					return next.Function, fmt.Sprintf("%s:%d", next.File, next.Line)
				}
			}

			// Fallback to the raw info.
			return frame.Function, fmt.Sprintf("%s:%d", frame.File, frame.Line)
		}
	}

	log.SetFormatter(formatter)
	log.Info("Log level set to ", lvl.String())
	return nil
}
