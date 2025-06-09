package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/etclab/art"
	"github.com/etclab/mu"
	"github.com/spf13/viper"
)

const usage = `Usage: akesod`

type Options struct {
	// optional
	setupRequired       bool
	setupTopic          string
	updateTopic         string
	metadataUpdateTopic string
	project             string
	outform             string
	keytype             string
	artConfigFile       string
	numOfMembers        int
	initiator           string
	outDir              string
	sigFile             string
	msgFile             string
	treeStateFile       string
	privIKFile          string
	kdfSalt             []byte
	strategy            string
	maxReencryptions    int
	maxConcUpdates      int

	// positional
	bucket   string
	basePath string

	//derived
	encoding art.KeyEncoding // derived from outform
}

func printUsage() {
	fmt.Fprintf(os.Stdout, "%s", usage)
}

func parseOptions() *Options {
	var err error

	opts := Options{}

	// Reading from Config.yaml
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("config")

	err = viper.ReadInConfig()
	if err != nil {
		mu.Fatalf("Error reading config file: %v", err)
	}

	opts.project = viper.GetString("cloud.project_id")
	opts.bucket = viper.GetString("cloud.bucket")
	opts.setupTopic = viper.GetString("cloud.setup_topic")
	opts.updateTopic = viper.GetString("cloud.update_topic")
	opts.metadataUpdateTopic = viper.GetString("cloud.metadata_update_topic")
	opts.setupRequired = viper.GetBool("art.setup_required")
	opts.artConfigFile = viper.GetString("art.config_file")
	opts.numOfMembers = viper.GetInt("art.num_of_members")
	opts.initiator = viper.GetString("art.initiator")
	opts.keytype = viper.GetString("art.keytype")
	opts.outform = viper.GetString("art.outform")
	opts.outDir = viper.GetString("art.outdir")
	opts.sigFile = viper.GetString("art.sigfile")
	opts.msgFile = viper.GetString("art.msgfile")
	opts.treeStateFile = viper.GetString("art.tree_state_file")
	opts.privIKFile = viper.GetString("art.priv_IK_file")
	opts.kdfSalt = []byte(viper.GetString("art.kdf_salt"))
	opts.strategy = viper.GetString("art.strategy")
	opts.maxReencryptions = viper.GetInt("akesod.max_reencryptions")
	opts.maxConcUpdates = viper.GetInt("akesod.max_concurrent_updates")
	// Override from flags if given
	flag.Usage = printUsage

	flag.Parse()

	// ART related options
	opts.keytype = strings.ToLower(opts.keytype)
	opts.basePath = flag.Arg(0)

	if opts.keytype != "ik" && opts.keytype != "ek" && opts.keytype != "" {
		mu.Fatalf("error: -keytype invalid value %q (must be ik|ek or empty)", opts.keytype)
	}

	opts.outform = strings.ToLower(opts.outform)
	opts.encoding, err = art.StringToKeyEncoding(opts.outform)
	if err != nil {
		mu.Fatalf("error: %v", err)
	}

	return &opts
}
