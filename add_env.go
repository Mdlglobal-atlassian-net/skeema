package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/skeema/mycli"
)

func init() {
	summary := "Add a new named environment to an existing host directory"
	desc := `Modifies the .skeema file in an existing host directory to add a new named
environment. For example, if ` + "`" + `skeema init` + "`" + ` was previously used to create a dir
for a host with the default "production" environment, ` + "`" + `skeema add-environment` + "`" + `
could be used to define a "staging" or "development" environment pointing at a different
host and port, or perhaps a "local" environment pointing at localhost and a
socket path.`

	cmd := mycli.NewCommand("add-environment", summary, desc, AddEnvHandler)
	cmd.AddOption(mycli.StringOption("host", 'h', "", "Database hostname or IP address"))
	cmd.AddOption(mycli.StringOption("port", 'P', "3306", "Port to use for database host"))
	cmd.AddOption(mycli.StringOption("socket", 'S', "/tmp/mysql.sock", "Absolute path to Unix domain socket file for use when host is localhost"))
	cmd.AddOption(mycli.StringOption("dir", 'd', ".", "Base dir for this host's schemas"))
	cmd.AddArg("environment", "", true)
	CommandSuite.AddSubCommand(cmd)
}

func AddEnvHandler(cfg *mycli.Config) error {
	AddGlobalConfigFiles(cfg)

	dir, err := NewDir(cfg.Get("dir"), cfg)
	if err != nil {
		return err
	}
	if !dir.Exists() {
		return errors.New("In add-environment, --dir must refer to a directory that already exists")
	}
	if !dir.HasOptionFile() {
		return fmt.Errorf("Dir %s does not have an existing .skeema file! Can only use `skeema add-environment` on a dir previously created by `skeema init`", dir)
	}

	hostOptionFile, err := dir.OptionFile()
	if err != nil || hostOptionFile == nil {
		return fmt.Errorf("Unable to read .skeema file for %s: %s", dir, err)
	}

	environment := cfg.Get("environment")
	if environment == "" || strings.ContainsAny(environment, "[]\n\r") {
		return fmt.Errorf("Environment name \"%s\" is invalid", environment)
	}
	if hostOptionFile.HasSection(environment) {
		return fmt.Errorf("Environment name \"%s\" already defined in %s", environment, hostOptionFile.Path())
	}
	if !hostOptionFile.SomeSectionHasOption("host") {
		return errors.New("This command should be run against a --dir whose .skeema file already defines a host for another environment")
	}

	if !cfg.OnCLI("host") {
		return errors.New("`skeema add-environment` requires --host to be supplied on CLI")
	}
	inst, err := dir.FirstInstance()
	if err != nil {
		return err
	} else if inst == nil {
		return errors.New("Command line did not specify which instance to connect to")
	}

	hostOptionFile.SetOptionValue(environment, "host", inst.Host)
	if inst.Host == "localhost" && inst.SocketPath != "" {
		hostOptionFile.SetOptionValue(environment, "socket", inst.SocketPath)
	} else {
		hostOptionFile.SetOptionValue(environment, "port", strconv.Itoa(inst.Port))
	}
	if cfg.OnCLI("user") {
		hostOptionFile.SetOptionValue(environment, "user", cfg.Get("user"))
	}

	// Write the option file
	if err := hostOptionFile.Write(true); err != nil {
		return err
	}
	dir.Config.MarkDirty()

	fmt.Printf("Added environment [%s] to %s\n", environment, hostOptionFile.Path())
	return nil
}
