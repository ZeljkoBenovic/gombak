package ssh

import (
	"fmt"
	"io"
	"net"
	"os"
	"path"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SSH struct {
	cl *ssh.Client
}

type ClientOpts func(config *ssh.ClientConfig)

func WithInsecureKeyExchange() ClientOpts {
	return func(c *ssh.ClientConfig) {
		c.KeyExchanges = append(c.KeyExchanges, "diffie-hellman-group-exchange-sha256")
	}
}

func WithIgnoreHostKey() ClientOpts {
	return func(c *ssh.ClientConfig) {
		c.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	}
}

func WithPassword(pass string) ClientOpts {
	return func(c *ssh.ClientConfig) {
		c.Auth = []ssh.AuthMethod{ssh.Password(pass)}
	}
}

func NewSSH(user, host, port string, opts ...ClientOpts) (*SSH, error) {
	sshConf := &ssh.ClientConfig{}
	sshConf.SetDefaults()
	sshConf.User = user

	for _, f := range opts {
		f(sshConf)
	}

	cl, err := ssh.Dial("tcp", net.JoinHostPort(host, port), sshConf)
	if err != nil {
		return nil, fmt.Errorf("could not create new ssh client: %w", err)
	}

	return &SSH{
		cl: cl,
	}, nil
}

func (s *SSH) Run(cmd string) (string, error) {
	sess, err := s.cl.NewSession()
	if err != nil {
		return "", fmt.Errorf("could not create new ssh session: %w", err)
	}

	defer sess.Close()

	byteOut, err := sess.CombinedOutput(cmd)
	if err != nil {
		return "", err
	}

	return string(byteOut), nil
}

func (s *SSH) Close() error {
	return s.cl.Close()
}

func (s *SSH) Download(downloadFrom, downloadTo string) error {
	cl, err := sftp.NewClient(s.cl)
	if err != nil {
		return fmt.Errorf("could not create new sftp client: %w", err)
	}
	defer cl.Close()

	_, err = os.Stat(path.Dir(downloadTo))
	if os.IsNotExist(err) {
		if err = os.Mkdir(path.Dir(downloadTo), 0666); err != nil {
			return fmt.Errorf("could not create backup dir: %w", err)
		}
	}

	local, err := os.Create(downloadTo)
	if err != nil {
		return fmt.Errorf("could not create new file: %w", err)
	}
	defer local.Close()

	remote, err := cl.Open(downloadFrom)
	if err != nil {
		return fmt.Errorf("could not open remote file: %w", err)
	}
	defer remote.Close()

	if _, err = io.Copy(local, remote); err != nil {
		return err
	}

	return local.Sync()
}

func (s *SSH) Delete(fileName string) error {
	cl, err := sftp.NewClient(s.cl)
	if err != nil {
		return fmt.Errorf("could not create new sftp client: %w", err)
	}

	defer cl.Close()

	return cl.Remove(fileName)
}
