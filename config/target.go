package config

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
)

const (
	defaultNetworkInstanceName = "default"
)

type TargetConfig struct {
	Name          string        `json:"name,omitempty" mapstructure:"name,omitempty"`
	Address       string        `json:"address,omitempty" mapstructure:"address,omitempty"`
	DefaultNI     string        `json:"default-ni,omitempty" mapstructure:"default-ni,omitempty"`
	Insecure      *bool         `json:"insecure,omitempty" mapstructure:"insecure,omitempty"`
	SkipVerify    *bool         `json:"skip-verify,omitempty" mapstructure:"skip-verify,omitempty"`
	Username      *string       `json:"username,omitempty" mapstructure:"username,omitempty"`
	Password      *string       `json:"password,omitempty" mapstructure:"password,omitempty"`
	Timeout       time.Duration `json:"timeout,omitempty" mapstructure:"timeout,omitempty"`
	TLSCert       *string       `json:"tls-cert,omitempty" mapstructure:"tls-cert,omitempty"`
	TLSKey        *string       `json:"tls-key,omitempty" mapstructure:"tls-key,omitempty"`
	TLSCA         *string       `json:"tlsca,omitempty" mapstructure:"tlsca,omitempty"`
	TLSMinVersion string        `json:"tls-min-version,omitempty" mapstructure:"tls-min-version,omitempty"`
	TLSMaxVersion string        `json:"tls-max-version,omitempty" mapstructure:"tls-max-version,omitempty"`
	TLSVersion    string        `json:"tls-version,omitempty" mapstructure:"tls-version,omitempty"`
	Gzip          *bool         `json:"gzip,omitempty" mapstructure:"gzip,omitempty"`
	MaxRcvMsgSize int           `json:"max-rcv-msg-size,omitempty" mapstructure:"max-rcv-msg-size,omitempty"`
	// modify RPC session params
	// Redundancy  string `json:"redundancy,omitempty" mapstructure:"redundancy,omitempty"`
	// Persistence string `json:"persistence,omitempty" mapstructure:"persistence,omitempty"`
	// AckType     string `json:"ack-type,omitempty" mapstructure:"ack-type,omitempty"`
}

func (c *Config) GetTargets() (map[string]*TargetConfig, error) {
	targetsConfigs := make(map[string]*TargetConfig)
	if len(c.Address) > 0 {
		var err error
		for _, addr := range c.Address {
			tc := new(TargetConfig)
			err = c.parseAddress(tc, addr)
			if err != nil {
				return nil, fmt.Errorf("%q failed to parse address: %v", addr, err)
			}
			c.setTargetConfigDefaults(tc)
			targetsConfigs[tc.Name] = tc
			c.logger.Debugf("%q target-config: %s", addr, tc)
		}
		return targetsConfigs, nil
	}
	targetsMap := c.FileConfig.GetStringMap("targets")
	if len(targetsMap) == 0 {
		return nil, errors.New("no targets found")
	}
	for addr, t := range targetsMap {
		tc := new(TargetConfig)
		switch t := t.(type) {
		case map[string]interface{}:
			decoder, err := mapstructure.NewDecoder(
				&mapstructure.DecoderConfig{
					DecodeHook: mapstructure.StringToTimeDurationHookFunc(),
					Result:     tc,
				},
			)
			if err != nil {
				return nil, err
			}
			err = decoder.Decode(t)
			if err != nil {
				return nil, err
			}
		case nil:
		default:
			return nil, fmt.Errorf("unexpected targets format, got a %T", t)
		}
		err := c.parseAddress(tc, addr)
		if err != nil {
			return nil, fmt.Errorf("%q failed to parse address: %v", addr, err)
		}
		c.setTargetConfigDefaults(tc)
		targetsConfigs[tc.Name] = tc
		c.logger.Debugf("%q target-config: %s", addr, tc)
	}
	return targetsConfigs, nil
}

func (c *Config) parseAddress(tc *TargetConfig, addr string) error {
	_, _, err := net.SplitHostPort(addr)
	if err != nil {
		if strings.Contains(err.Error(), "missing port in address") ||
			strings.Contains(err.Error(), "too many colons in address") {
			tc.Address = net.JoinHostPort(addr, c.Port)
			return nil
		}
		return fmt.Errorf("error parsing address %q: %v", addr, err)

	}
	tc.Address = addr
	return nil
}

func (c *Config) setTargetConfigDefaults(tc *TargetConfig) {
	if tc.Name == "" {
		tc.Name = tc.Address
	}
	if tc.DefaultNI == "" {
		tc.DefaultNI = defaultNetworkInstanceName
	}
	if c.Insecure {
		tc.Insecure = &c.Insecure
	}
	if tc.Timeout <= 0 {
		tc.Timeout = c.Timeout
	}
	if tc.Username == nil {
		tc.Username = &c.Username
	}
	if tc.Password == nil {
		tc.Password = &c.Password
	}
	if tc.SkipVerify == nil {
		tc.SkipVerify = &c.SkipVerify
	}
	if tc.Insecure == nil || (tc.Insecure != nil && !*tc.Insecure) {
		if tc.TLSCA == nil {
			if c.TLSCa != "" {
				tc.TLSCA = &c.TLSCa
			}
		}
		if tc.TLSCert == nil {
			tc.TLSCert = &c.TLSCert
		}
		if tc.TLSKey == nil {
			tc.TLSKey = &c.TLSKey
		}
	}
	if tc.TLSVersion == "" {
		tc.TLSVersion = c.TLSVersion
	}
	if tc.TLSMinVersion == "" {
		tc.TLSMinVersion = c.TLSMinVersion
	}
	if tc.TLSMaxVersion == "" {
		tc.TLSMaxVersion = c.TLSMaxVersion
	}
	if tc.Gzip == nil {
		tc.Gzip = &c.Gzip
	}
	if tc.MaxRcvMsgSize == 0 {
		tc.MaxRcvMsgSize = c.MaxRcvMsgSize
	}
}

func (tc *TargetConfig) DialOpts() ([]grpc.DialOption, error) {
	tOpts := make([]grpc.DialOption, 0)
	if tc.Insecure != nil && *tc.Insecure {
		tOpts = append(tOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		tlsConfig, err := tc.newTLS()
		if err != nil {
			return nil, err
		}
		tOpts = append(tOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	}
	if tc.Gzip != nil && *tc.Gzip {
		tOpts = append(tOpts, grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)))
	}
	if tc.MaxRcvMsgSize > 0 {
		tOpts = append(tOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(tc.MaxRcvMsgSize)))
	}
	return tOpts, nil
}

func (tc *TargetConfig) newTLS() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		Renegotiation:      tls.RenegotiateNever,
		InsecureSkipVerify: *tc.SkipVerify,
		MaxVersion:         tc.getTLSMaxVersion(),
		MinVersion:         tc.getTLSMinVersion(),
	}
	err := loadCerts(tlsConfig, tc)
	if err != nil {
		return nil, err
	}
	return tlsConfig, nil
}

func (tc *TargetConfig) getTLSMinVersion() uint16 {
	v := tlsVersionStringToUint(tc.TLSVersion)
	if v > 0 {
		return v
	}
	return tlsVersionStringToUint(tc.TLSMinVersion)
}

func (tc *TargetConfig) getTLSMaxVersion() uint16 {
	v := tlsVersionStringToUint(tc.TLSVersion)
	if v > 0 {
		return v
	}
	return tlsVersionStringToUint(tc.TLSMaxVersion)
}

func tlsVersionStringToUint(v string) uint16 {
	switch v {
	default:
		return 0
	case "1.3":
		return tls.VersionTLS13
	case "1.2":
		return tls.VersionTLS12
	case "1.1":
		return tls.VersionTLS11
	case "1.0", "1":
		return tls.VersionTLS10
	}
}

func loadCerts(tlscfg *tls.Config, tc *TargetConfig) error {
	if *tc.TLSCert != "" && *tc.TLSKey != "" {
		certificate, err := tls.LoadX509KeyPair(*tc.TLSCert, *tc.TLSKey)
		if err != nil {
			return err
		}
		tlscfg.Certificates = []tls.Certificate{certificate}
		// tlscfg.BuildNameToCertificate()
	}
	if tc.TLSCA != nil && *tc.TLSCA != "" {
		certPool := x509.NewCertPool()
		caFile, err := ioutil.ReadFile(*tc.TLSCA)
		if err != nil {
			return err
		}
		if ok := certPool.AppendCertsFromPEM(caFile); !ok {
			return errors.New("failed to append certificate")
		}
		tlscfg.RootCAs = certPool
	}
	return nil
}

func (tc *TargetConfig) String() string {
	b, err := json.Marshal(tc)
	if err != nil {
		return ""
	}
	return string(b)
}
