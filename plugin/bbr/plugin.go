package main

import (
	"archive/tar"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	fmt "github.com/jhunt/go-ansi"
	"github.com/starkandwayne/shield/plugin"
)

const (
	DefaultBinDir = "/var/vcap/packages/bbr/bin"
)

func main() {
	bbr := BBRPlugin{
		Name:    "BOSH Backup & Restore (BBR) Plugin",
		Author:  "Stark & Wayne",
		Version: "1.0.0",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
		Example: `
{
  "bindir"     : "/path/to/bbr-package/bin",
  "username"   : "admin",
  "password"   : "c1oudc0w",
  "director"   : "192.168.50.6",
  "deployment" : "cf",
  "cacert"     : "-----BEGIN CERTIFICATE-----\n(cert contents)\n(... etc ...)\n-----END CERTIFICATE-----"
}
`,
		Defaults: `
{
  "bindir"   : "/var/vcap/packages/bbr/bin",
}
`,
		Fields: []plugin.Field{
			plugin.Field{
				Mode:     "target",
				Name:     "director",
				Type:     "string",
				Title:    "BOSH Director",
				Help:     "The hostname or IP address of your BOSH Director.",
				Required: true,
			},
			plugin.Field{
				Mode:  "target",
				Name:  "deployment",
				Type:  "string",
				Title: "Deployment",
				Help:  "The name of the BOSH deployment to back up.  Leave blank for director backups.",
			},
			plugin.Field{
				Mode:  "target",
				Name:  "ca_cert",
				Type:  "text",
				Title: "CA Certificate",
				Help:  "Certificate Authority X.509 Certificate that signed the BOSH Director's TLS certificate.",
			},
			plugin.Field{
				Mode:  "target",
				Name:  "bosh_username",
				Type:  "string",
				Title: "BOSH Username",
				Help:  "Username for authenticating to the BOSH Director.",
			},
			plugin.Field{
				Mode:  "target",
				Name:  "bosh_password",
				Type:  "password",
				Title: "BOSH Password",
				Help:  "Password for authenticating to the BOSH Director.",
			},
			plugin.Field{
				Mode:  "target",
				Name:  "system_username",
				Type:  "string",
				Title: "VM Username",
				Help:  "Username to SSH to the BOSH Director as (director backups only).",
			},
			plugin.Field{
				Mode:  "target",
				Name:  "system_key",
				Type:  "text",
				Title: "VM Private Key",
				Help:  "RSA Private Key for the System user.",
			},
			plugin.Field{
				Mode:    "target",
				Name:    "bindir",
				Type:    "abspath",
				Title:   "BBR bin/ Path",
				Help:    "The absolute path to the bin/ directory that contains the `bbr` command.",
				Default: "/var/vcap/packages/bbr/bin",
			},
			plugin.Field{
				Mode:    "target",
				Name:    "tmpdir",
				Type:    "abspath",
				Title:   "Temporary Directory",
				Help:    "The absolute path to a temporary directory (like /tmp) in which to stage backup files.",
				Default: "",
			},
		},
	}

	fmt.Fprintf(os.Stderr, "bbr plugin starting up...\n")
	plugin.Run(bbr)
}

type BBRPlugin plugin.PluginInfo

type details struct {
	BinDir         string
	TempDir        string
	BOSHUsername   string
	BOSHPassword   string
	SystemUsername string
	SystemKey      string
	Director       string
	Deployment     string
	CACert         string
}

func (p BBRPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p BBRPlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	fail := false

	deployment, _ := endpoint.StringValueDefault("deployment", "")
	if deployment != "" {
		fmt.Printf("bbr plugin operating in @G{deployment} mode...\n")

		if s, err := endpoint.StringValueDefault("bindir", DefaultBinDir); err != nil {
			fmt.Printf("@R{\u2717 bindir           %s}\n", err)
			fail = true
		} else {
			fmt.Printf("@G{\u2713 bindir}           @C{%s}\n", s)
		}

		if s, err := endpoint.StringValueDefault("tmpdir", os.TempDir()); err != nil {
			fmt.Printf("@R{\u2717 tmpdir           %s}\n", err)
			fail = true
		} else {
			fmt.Printf("@G{\u2713 tmpdir}           @C{%s}\n", s)
		}

		if s, err := endpoint.StringValue("director"); err != nil {
			fmt.Printf("@R{\u2717 director         %s}\n", err)
			fail = true
		} else {
			fmt.Printf("@G{\u2713 director}         @C{%s}\n", s)
		}

		if s, err := endpoint.StringValue("ca_cert"); err != nil {
			fmt.Printf("@R{\u2717 ca_cert          %s}\n", err)
			fail = true
		} else {
			/* FIXME: validate that it is an X.509 PEM certificate */
			lines := strings.Split(s, "\n")
			fmt.Printf("@G{\u2713 ca_cert}          @C{%s}\n", lines[0])
			if len(lines) > 1 {
				for _, line := range lines[1:] {
					fmt.Printf("                   @C{%s}\n", line)
				}
			}
		}

		if s, err := endpoint.StringValue("bosh_username"); err != nil {
			fmt.Printf("@R{\u2717 bosh_username    %s}\n", err)
			fail = true
		} else {
			fmt.Printf("@G{\u2713 bosh_username}    @C{%s}\n", s)
		}

		if s, err := endpoint.StringValue("bosh_password"); err != nil {
			fmt.Printf("@R{\u2717 bosh_password    %s}\n", err)
			fail = true
		} else {
			fmt.Printf("@G{\u2713 bosh_password}    @C{%s}\n", plugin.Redact(s))
		}

		if s, err := endpoint.StringValue("deployment"); err != nil {
			fmt.Printf("@R{\u2717 deployment       %s}\n", err)
			fail = true
		} else {
			fmt.Printf("@G{\u2713 deployment}       @C{%s}\n", s)
		}

	} else {
		fmt.Printf("bbr plugin operating in @G{director} mode...\n")

		if s, err := endpoint.StringValueDefault("bindir", DefaultBinDir); err != nil {
			fmt.Printf("@R{\u2717 bindir           %s}\n", err)
			fail = true
		} else {
			fmt.Printf("@G{\u2713 bindir}           @C{%s}\n", s)
		}

		if s, err := endpoint.StringValue("director"); err != nil {
			fmt.Printf("@R{\u2717 director         %s}\n", err)
			fail = true
		} else {
			fmt.Printf("@G{\u2713 director}         @C{%s}\n", s)
		}

		if s, err := endpoint.StringValue("system_username"); err != nil {
			fmt.Printf("@R{\u2717 system_username  %s}\n", err)
			fail = true
		} else {
			fmt.Printf("@G{\u2713 system_username}  @C{%s}\n", s)
		}

		if s, err := endpoint.StringValue("system_key"); err != nil {
			fmt.Printf("@R{\u2717 system_key       %s}\n", err)
			fail = true
		} else {
			/* FIXME: validate that it's an RSA formatted key */
			lines := strings.Split(s, "\n")
			fmt.Printf("@G{\u2713 system_key}       <redacted>@C{%s}\n", lines[0])
			if len(lines) > 1 {
				for _, line := range lines[1:] {
					fmt.Printf("                  @C{%s}\n", line)
				}
			}
			fmt.Printf("</redacted>\n")
		}

	}

	if fail {
		return fmt.Errorf("bbr: invalid configuration")
	}
	return nil
}

func persist(dir, contents string) (string, error) {
	f, err := ioutil.TempFile("", "shield-bbr-*")
	if err != nil {
		return "", err
	}

	if _, err := f.WriteString(contents); err != nil {
		return "", err
	}

	if err := f.Close(); err != nil {
		return "", err
	}

	return f.Name(), nil
}

func (p BBRPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	var cmd *exec.Cmd

	bbr, err := getDetails(endpoint)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Setting up temporary workspace directory...\n")
	workspace, err := ioutil.TempDir(bbr.TempDir, "shield-bbr-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workspace)
	fmt.Fprintf(os.Stderr, "  workspace is '%s'\n", workspace)

	fmt.Fprintf(os.Stderr, "Changing working directory to workspace...\n")
	if err := os.Chdir(workspace); err != nil {
		return err
	}

	if bbr.Deployment != "" {
		fmt.Fprintf(os.Stderr, "\nstarting BBR backup in deployment mode...\n")

		fmt.Fprintf(os.Stderr, "Writing CA Certificate file to disk...\n")
		ca, err := persist(bbr.TempDir, bbr.CACert)
		if err != nil {
			return err
		}
		defer os.Remove(ca)
		fmt.Fprintf(os.Stderr, "  wrote to '%s'\n", ca)

		cmd = exec.Command(fmt.Sprintf("%s/bbr", bbr.BinDir),
			"deployment",
			"--target", bbr.Director,
			"--username", bbr.BOSHUsername,
			"--password", bbr.BOSHPassword,
			"--deployment", bbr.Deployment,
			"--ca-cert", ca,
			"backup")
		fmt.Fprintf(os.Stderr, "\nRunning BRR CLI...\n")
		fmt.Fprintf(os.Stderr, "  %s/bbr deployment \\\n", bbr.BinDir)
		fmt.Fprintf(os.Stderr, "    --target %s \\\n", bbr.Director)
		fmt.Fprintf(os.Stderr, "    --username %s \\\n", bbr.BOSHUsername)
		fmt.Fprintf(os.Stderr, "    --password %s \\\n", plugin.Redact(bbr.BOSHPassword))
		fmt.Fprintf(os.Stderr, "    --deployment %s \\\n", bbr.Deployment)
		fmt.Fprintf(os.Stderr, "    --ca-cert %s \\\n", ca)
		fmt.Fprintf(os.Stderr, "    backup\n\n\n")

	} else {
		fmt.Fprintf(os.Stderr, "\nstarting BBR backup in director mode...\n")
		fmt.Fprintf(os.Stderr, "Writing SSH Private Key to disk...\n")
		key, err := persist(bbr.TempDir, bbr.SystemKey)
		if err != nil {
			return err
		}
		defer os.Remove(key)
		fmt.Fprintf(os.Stderr, "  wrote to '%s'\n", key)

		cmd = exec.Command(fmt.Sprintf("%s/bbr", bbr.BinDir),
			"director",
			"--host", bbr.Director,
			"--username", bbr.SystemUsername,
			"--private-key-path", key,
			"backup")

		fmt.Fprintf(os.Stderr, "\nRunning BRR CLI...\n")
		fmt.Fprintf(os.Stderr, "  %s/bbr director \\\n", bbr.BinDir)
		fmt.Fprintf(os.Stderr, "    --host %s \\\n", bbr.Director)
		fmt.Fprintf(os.Stderr, "    --username %s \\\n", bbr.SystemUsername)
		fmt.Fprintf(os.Stderr, "    --private-key-path %s \\\n", key)
		fmt.Fprintf(os.Stderr, "    backup\n\n\n")
	}

	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	fmt.Fprintf(os.Stderr, "----------------------------------------------------------\n")
	err = cmd.Run()
	fmt.Fprintf(os.Stderr, "----------------------------------------------------------\n\n\n")
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Combining BBR output files into an uncompressed tarball...\n")
	archive := tar.NewWriter(os.Stdout)
	err = filepath.Walk(workspace, func(path string, info os.FileInfo, err error) error {
		fmt.Fprintf(os.Stderr, "  - analyzing %s ... ", path)
		rel, err := filepath.Rel(workspace, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{FAILED}\n")
			return err
		}

		if !strings.HasPrefix(rel, bbr.Deployment) {
			fmt.Fprintf(os.Stderr, "skipping\n")
			return nil
		}

		fmt.Fprintf(os.Stderr, "INCLUDING\n")
		h, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		h.Name = rel
		archive.WriteHeader(h)

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		io.Copy(archive, f)
		f.Close()

		return nil
	})
	if err != nil {
		return err
	}

	archive.Close()
	return nil
}

func (p BBRPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	var cmd *exec.Cmd

	bbr, err := getDetails(endpoint)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Setting up temporary workspace directory...\n")
	workspace, err := ioutil.TempDir(bbr.TempDir, "shield-bbr-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workspace)
	fmt.Fprintf(os.Stderr, "  workspace is '%s'\n", workspace)

	fmt.Fprintf(os.Stderr, "Changing working directory to workspace...\n")
	if err := os.Chdir(workspace); err != nil {
		return err
	}

	archive := tar.NewReader(os.Stdin)
	for {
		h, err := archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		f, err := os.Create(h.Name)
		if err != nil {
			return err
		}
		io.Copy(f, archive)
		f.Close()
	}

	artifacts, err := filepath.Glob("*")
	if err != nil {
		return err
	}
	if len(artifacts) == 0 {
		return fmt.Errorf("Found no BBR artifacts in backup archive!")
	}
	if len(artifacts) > 1 {
		return fmt.Errorf("Found too many BBR artifacts (%d) in backup archive!", len(artifacts))
	}

	if bbr.Deployment != "" {
		fmt.Fprintf(os.Stderr, "\nstarting BBR restore in deployment mode...\n")

		fmt.Fprintf(os.Stderr, "Writing CA Certificate file to disk...\n")
		ca, err := persist(bbr.TempDir, bbr.CACert)
		if err != nil {
			return err
		}
		defer os.Remove(ca)
		fmt.Fprintf(os.Stderr, "  wrote to '%s'\n", ca)

		cmd = exec.Command(fmt.Sprintf("%s/bbr", bbr.BinDir),
			"deployment",
			"--target", bbr.Director,
			"--username", bbr.BOSHUsername,
			"--password", bbr.BOSHPassword,
			"--deployment", bbr.Deployment,
			"--ca-cert", ca,
			"restore",
			"--artifact-path", artifacts[0])

		fmt.Fprintf(os.Stderr, "\nRunning BRR CLI...\n")
		fmt.Fprintf(os.Stderr, "  %s/bbr deployment \\\n", bbr.BinDir)
		fmt.Fprintf(os.Stderr, "    --target %s \\\n", bbr.Director)
		fmt.Fprintf(os.Stderr, "    --username %s \\\n", bbr.BOSHUsername)
		fmt.Fprintf(os.Stderr, "    --password %s \\\n", plugin.Redact(bbr.BOSHPassword))
		fmt.Fprintf(os.Stderr, "    --deployment %s \\\n", bbr.Deployment)
		fmt.Fprintf(os.Stderr, "    --ca-cert %s \\\n", ca)
		fmt.Fprintf(os.Stderr, "    restore \\\n")
		fmt.Fprintf(os.Stderr, "    --artifact-path %s \\\n\n\n", artifacts[0])

	} else {
		fmt.Fprintf(os.Stderr, "\nstarting BBR backup in director mode...\n")
		fmt.Fprintf(os.Stderr, "Writing SSH Private Key to disk...\n")
		key, err := persist(bbr.TempDir, bbr.SystemKey)
		if err != nil {
			return err
		}
		defer os.Remove(key)
		fmt.Fprintf(os.Stderr, "  wrote to '%s'\n", key)

		cmd = exec.Command(fmt.Sprintf("%s/bbr", bbr.BinDir),
			"director",
			"--host", bbr.Director,
			"--username", bbr.SystemUsername,
			"--private-key-path", key,
			"restore",
			"--artifact-path", artifacts[0])

		fmt.Fprintf(os.Stderr, "\nRunning BRR CLI...\n")
		fmt.Fprintf(os.Stderr, "  %s/bbr director \\\n", bbr.BinDir)
		fmt.Fprintf(os.Stderr, "    --host %s \\\n", bbr.Director)
		fmt.Fprintf(os.Stderr, "    --username %s \\\n", bbr.SystemUsername)
		fmt.Fprintf(os.Stderr, "    --private-key-path %s \\\n", key)
		fmt.Fprintf(os.Stderr, "    restore \\\n")
		fmt.Fprintf(os.Stderr, "    --artifact-path %s \\\n\n\n", artifacts[0])
	}

	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	fmt.Fprintf(os.Stderr, "----------------------------------------------------------\n")
	err = cmd.Run()
	fmt.Fprintf(os.Stderr, "----------------------------------------------------------\n\n\n")
	return err
}

func (p BBRPlugin) Store(endpoint plugin.ShieldEndpoint) (string, int64, error) {
	return "", 0, plugin.UNIMPLEMENTED
}

func (p BBRPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p BBRPlugin) Purge(endpoint plugin.ShieldEndpoint, key string) error {
	return plugin.UNIMPLEMENTED
}

func getDetails(endpoint plugin.ShieldEndpoint) (*details, error) {
	deployment, err := endpoint.StringValueDefault("deployment", "")
	if err != nil {
		return nil, err
	}

	director, err := endpoint.StringValue("director")
	if err != nil {
		return nil, err
	}

	bin, err := endpoint.StringValueDefault("bindir", DefaultBinDir)
	if err != nil {
		return nil, err
	}

	tmp, err := endpoint.StringValueDefault("tmpdir", os.TempDir())
	if err != nil {
		return nil, err
	}

	if deployment != "" {
		cacert, err := endpoint.StringValue("ca_cert")
		if err != nil {
			return nil, err
		}

		username, err := endpoint.StringValue("bosh_username")
		if err != nil {
			return nil, err
		}

		password, err := endpoint.StringValue("bosh_password")
		if err != nil {
			return nil, err
		}

		return &details{
			BinDir:       bin,
			TempDir:      tmp,
			Director:     director,
			BOSHUsername: username,
			BOSHPassword: password,
			CACert:       cacert,
			Deployment:   deployment,
		}, nil
	}

	username, err := endpoint.StringValue("system_username")
	if err != nil {
		return nil, err
	}

	key, err := endpoint.StringValue("system_key")
	if err != nil {
		return nil, err
	}

	return &details{
		BinDir:         bin,
		TempDir:        tmp,
		Director:       director,
		SystemUsername: username,
		SystemKey:      key,
	}, nil
}
