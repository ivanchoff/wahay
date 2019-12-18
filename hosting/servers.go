package hosting

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/digitalautonomy/grumble/pkg/logtarget"
	grumbleServer "github.com/digitalautonomy/grumble/server"
)

// Servers serves
type Servers interface {
	CreateServer(port string) (Server, error)
	DestroyServer(Server) error
	Shutdown() error
}

// Create creates
func Create() (Servers, error) {
	s := &servers{}
	e := s.create()
	return s, e
}

type servers struct {
	dataDir string
	started bool
	servers map[int64]*grumbleServer.Server
	log     *log.Logger
}

func (s *servers) initializeSharedObjects() {
	s.servers = make(map[int64]*grumbleServer.Server)
	grumbleServer.SetServers(s.servers)
}

func (s *servers) initializeDataDirectory() error {
	var e error
	s.dataDir, e = ioutil.TempDir("", "tonio")
	if e != nil {
		return e
	}

	grumbleServer.Args.DataDir = s.dataDir

	serversDirPath := filepath.Join(s.dataDir, "servers")
	e = os.Mkdir(serversDirPath, 0700)
	if e != nil && !os.IsExist(e) {
		return e
	}

	return nil
}

func (s *servers) initializeLogging() error {
	logDir := path.Join(s.dataDir, "grumble.log")
	grumbleServer.Args.LogPath = logDir

	err := logtarget.Target.OpenFile(logDir)
	if err != nil {
		return err
	}

	s.log = log.New(&logtarget.Target, "[G] ", log.LstdFlags|log.Lmicroseconds)
	s.log.Printf("Grumble")
	s.log.Printf("Using data directory: %s", s.dataDir)

	return nil
}

func (s *servers) initializeCertificates() error {
	s.log.Printf("Generating 4096-bit RSA keypair for self-signed certificate...")

	certFn := filepath.Join(s.dataDir, "cert.pem")
	keyFn := filepath.Join(s.dataDir, "key.pem")
	err := grumbleServer.GenerateSelfSignedCert(certFn, keyFn)
	if err != nil {
		return err
	}

	s.log.Printf("Certificate output to %v", certFn)
	s.log.Printf("Private key output to %v", keyFn)
	return nil
}

// create will initialize all grumble things
// because the grumble server package uses global
// state it is NOT advisable to call this function
// more than once in a program
func (s *servers) create() error {
	s.initializeSharedObjects()

	e := s.initializeDataDirectory()
	if e != nil {
		return e
	}

	e = s.initializeLogging()
	if e != nil {
		return e
	}

	e = s.initializeCertificates()
	if e != nil {
		return e
	}

	return nil
}

func (s *servers) startListener() {
	if !s.started {
		go grumbleServer.SignalHandler()
		s.started = true
	}
}

func (s *servers) CreateServer(port string) (Server, error) {
	nextID := len(s.servers) + 1
	serv, err := grumbleServer.NewServer(int64(nextID))
	if err != nil {
		return nil, err
	}
	s.servers[serv.Id] = serv
	serv.Set("NoWebServer", "true")
	serv.Set("Address", "127.0.0.1")
	serv.Set("Port", port)

	err = os.Mkdir(filepath.Join(s.dataDir, "servers", fmt.Sprintf("%v", 1)), 0750)
	if err != nil {
		return nil, err
	}

	return &server{s, serv}, nil
}

func (s *servers) DestroyServer(Server) error {
	// For now, this function will do nothing. We will still call it,
	// in case we need it in the server
	return nil
}

func (s *servers) Shutdown() error {
	return os.RemoveAll(s.dataDir)
}