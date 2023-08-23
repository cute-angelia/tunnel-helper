package internal

import (
	"encoding/json"
	"log"
	"os"
)

const (
	defaultCfgName = "config.json"
)

func CheckFileExist() {
	if _, err := os.Stat("config.json"); os.IsNotExist(err) {
		// 文件不存在
		log.Println("配置文件不存在，创建新的配置文件，请修改对应信息")
		generateConfig()
	}
}

func generateConfig() error {
	var defaultConfig = Config{
		SSH: &SSHConfig{
			Host:           "172.168.1.1",
			User:           "username",
			Port:           22,
			Secret:         "SSH-PASSWORD",
			PrivateKeyFile: "/path/to/.ssh/id_rsa",
		},
		Tunnels: []*TunnelConfig{
			&TunnelConfig{
				Ident:      "tunnel-config-without-ssh",
				SSH:        nil,
				LocalPort:  8081,
				RemoteHost: "172.168.1.1",
				RemotePort: 8081,
			},
			&TunnelConfig{
				Ident: "tunnel-config-with-ssh",
				SSH: &SSHConfig{
					Host:           "172.168.1.2",
					User:           "username2",
					Port:           22,
					Secret:         "SSH-PASSWORD2",
					PrivateKeyFile: "/path/to/.ssh/id_rsa",
				},
				LocalPort:  8080,
				RemoteHost: "172.168.1.1",
				RemotePort: 8080,
			},
		},
	}
	fd, err := os.OpenFile(defaultCfgName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Panicf("could not open file=%s, err=%v", defaultCfgName, err)
		return err
	}

	defer fd.Close()

	byts, err := json.MarshalIndent(defaultConfig, "", "\t")
	if err != nil {
		log.Panicf("could not marshal defaultConfig=%+v, err=%v", defaultConfig, err)
		return err
	}

	_, err = fd.Write(byts)
	return err
}
