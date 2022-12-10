package gocert

import (
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
	"strings"
	"time"
)

const SubjectKey = "gocert.generate"

func GenerateNats(domain string, isClient bool) (*CertificateResponse, error) {
	if !strings.Contains(domain, ".") {
		return nil, fmt.Errorf("domain must contain at least one dot")
	}

	conn, _ := nats.Connect(
		"nats://10.0.0.140:4222",
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)

	defer conn.Close()

	resp, err := conn.Request(SubjectKey, []byte(fmt.Sprintf("%s/%t", domain, isClient)), 5*time.Second)
	if err != nil {
		return nil, err
	}
	cr := &CertificateResponse{}
	err = json.Unmarshal(resp.Data, cr)
	return cr, err
}
