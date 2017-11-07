package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

type Consul struct {
	addr   *string
	token  *string
	Client *api.Client
}

type Config struct {
	env     *string
	project *string
	userEnv *string
	consul  *Consul
	file    *string
}

type KVPair struct {
	key   string
	value string
}

func main() {
	conf := &Config{}
	consul := &Consul{}
	consul.token = flag.StringP("token", "t", os.Getenv("CONSUL_TOKEN"), "Consul token")
	consul.addr = flag.String("addr", os.Getenv("CONSUL_URL"), "Consul URL")
	conf.env = flag.StringP("env", "e", "dev", "Main environment")
	conf.project = flag.StringP("project", "p", "", "Project name")
	conf.userEnv = flag.StringP("userenv", "u", "", "User environment")
	conf.file = flag.StringP("file", "f", "-", "Properties file (or - for stdin)")
	flag.Parse()
	consulClient, err := api.NewClient(consul.getConfig())
	if err != nil {
		log.Fatal("Unable to connect to consul server")
	}
	consul.Client = consulClient
	conf.consul = consul

	if *conf.file == "-" {
		err = conf.parseData(bufio.NewScanner(os.Stdin))
	} else {
		inFile, err := os.Open(*conf.file)
		if err != nil {
			log.Fatal("Unable to open file")
		}
		defer inFile.Close()
		err = conf.parseData(bufio.NewScanner(inFile))
	}

}

func (conf *Config) Clean(line, prefix string) *api.KVPair {
	cleanLine := strings.TrimSpace(line)
	if cleanLine == "" {
		return nil
	}
	re := regexp.MustCompile(`^(!|#)`)
	if re.MatchString(cleanLine) {
		return nil
	}
	splitArr := strings.SplitN(cleanLine, "=", 2)
	comments := regexp.MustCompile(`^[^#]*`)
	return &api.KVPair{
		Key:   fmt.Sprintf("%s/%s", prefix, strings.Replace(splitArr[0], ".", "/", -1)),
		Value: bytes.TrimSpace(comments.Find([]byte(splitArr[1]))),
	}
}

func (conf *Config) parseData(buf *bufio.Scanner) error {
	kv := conf.consul.Client.KV()
	path := fmt.Sprintf("config/%s/%s/%s", *conf.env, *conf.project, *conf.userEnv)
	kv.DeleteTree(path, nil)

	for buf.Scan() {
		if t := conf.Clean(buf.Text(), path); t != nil {
			if _, err := kv.Put(t, nil); err != nil {
				fmt.Println(err)
			}
			fmt.Println(t)
		}
	}
	return nil
}

func (c *Consul) getConfig() *api.Config {
	defaultConf := api.DefaultConfig()
	defaultConf.Token = *c.token
	defaultConf.Address = *c.addr
	defaultConf.Scheme = "http"
	return defaultConf
}
