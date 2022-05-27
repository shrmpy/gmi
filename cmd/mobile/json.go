package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)
import "github.com/shrmpy/gmi"

type Config struct {
	Title string

	TLS struct {
		MinimumVersion   string
		SelfSigned       gmi.Mask
		LegacyCommonName gmi.Mask
		Expired          gmi.Mask
	}
	Gemini struct {
		FollowRedirect int
		WrapText       string
	}
	Log struct {
		Level string
	}
}

func maskFrom(cfg *Config) gmi.Mask {
	var isv gmi.Mask
	if cfg == nil {
		log.Printf("INFO skipped isv, empty config")
		return isv
	}
	if cfg.TLS.LegacyCommonName.Has(gmi.AcceptLCN) ||
		cfg.TLS.LegacyCommonName.Has(gmi.LCNPrompt) ||
		cfg.TLS.LegacyCommonName.Has(gmi.LCNReject) {
		log.Printf("INFO isv with LCN bit, %v", cfg.TLS.LegacyCommonName)
		isv = isv.Set(cfg.TLS.LegacyCommonName)
	}
	if cfg.TLS.SelfSigned.Has(gmi.AcceptUAE) ||
		cfg.TLS.SelfSigned.Has(gmi.PromptUAE) ||
		cfg.TLS.SelfSigned.Has(gmi.UAEReject) {
		log.Printf("INFO isv with UAE bit, %v", cfg.TLS.SelfSigned)
		isv = isv.Set(cfg.TLS.SelfSigned)
	}
	if cfg.TLS.Expired.Has(gmi.AcceptCIE) ||
		cfg.TLS.Expired.Has(gmi.CIEPrompt) ||
		cfg.TLS.Expired.Has(gmi.CIEReject) {
		log.Printf("INFO isv with EXC bit, %v", cfg.TLS.Expired)
		isv = isv.Set(cfg.TLS.Expired)
	}

	return isv
}
func readArgs() (*Config, error) {
	var (
		err error
		cfg *Config
		js  = flag.String("json", "config.json", "JSON file path")
	)
	flag.Parse()
	log.SetFlags(log.Lshortfile | log.Ltime)

	if cfg, err = readConfig(*js); err == nil {
		return cfg, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return safeConfig(), nil
	}
	return nil, err
}

func readConfig(filename string) (*Config, error) {
	var (
		err  error
		buf  []byte
		abs  string
		cfg  Config
		data map[string]interface{}
	)
	if abs, err = filepath.Abs(filename); err != nil {
		return nil, err
	}
	if buf, err = os.ReadFile(abs); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(buf, &data); err != nil {
		return nil, err
	}
	cfg = hydrate(data)
	return &cfg, nil
}
func safeConfig() *Config {
	var c = Config{Title: "Safe defaults"}
	c.TLS.MinimumVersion = "1.2"
	c.TLS.SelfSigned = gmi.PromptUAE
	c.TLS.LegacyCommonName = gmi.AcceptLCN
	c.TLS.Expired = gmi.CIEReject
	c.Gemini.FollowRedirect = 0
	c.Gemini.WrapText = "none"
	c.Log.Level = "verbose"
	return &c
}
func hydrate(data map[string]interface{}) Config {
	var tmp = Config{Title: "empty"}
	if dtls, ok := data["tls"]; ok {
		if mtls, ok := dtls.(map[string]interface{}); ok {
			if ex, ok := mtls["expired"]; ok {
				if na, ok := ex.(string); ok {
					tmp.TLS.Expired = toMask(na)
				}
			}
			if se, ok := mtls["self_signed"]; ok {
				if na, ok := se.(string); ok {
					tmp.TLS.SelfSigned = toMask(na)
				}
			}
			if le, ok := mtls["legacy_common_name"]; ok {
				if na, ok := le.(string); ok {
					tmp.TLS.LegacyCommonName = toMask(na)
				}
			}
			if mv, ok := mtls["minimum_version"]; ok {
				if ver, ok := mv.(string); ok {
					tmp.TLS.MinimumVersion = ver
				}
			}
		}
	}
	if dgem, ok := data["gemini"]; ok {
		if mgem, ok := dgem.(map[string]interface{}); ok {
			if re, ok := mgem["follow_redirect"]; ok {
				if mx, ok := re.(float64); ok {
					tmp.Gemini.FollowRedirect = int(mx)
				}
			}
			if wr, ok := mgem["wrap_text"]; ok {
				if na, ok := wr.(string); ok {
					tmp.Gemini.WrapText = na
				}
			}
		}
	}
	if dlog, ok := data["log"]; ok {
		if mlog, ok := dlog.(map[string]interface{}); ok {
			if lv, ok := mlog["level"]; ok {
				if na, ok := lv.(string); ok {
					tmp.Log.Level = na
				}
			}
		}
	}
	if ti, ok := data["title"]; ok {
		if na, ok := ti.(string); ok {
			tmp.Title = na
		}
	}

	log.Printf("DEBUG hydrate from json data, %v", tmp)
	return tmp
}
func toMask(name string) gmi.Mask {
	switch strings.ToLower(name) {
	case "sscreject":
		return gmi.UAEReject
	case "lcnreject":
		return gmi.LCNReject
	case "ciereject":
		return gmi.CIEReject
	case "promptssc":
		return gmi.PromptUAE
	case "lcnprompt":
		return gmi.LCNPrompt
	case "cieprompt":
		return gmi.CIEPrompt
	case "acceptssc":
		return gmi.AcceptUAE
	case "acceptlcn":
		return gmi.AcceptLCN
	case "acceptcie":
		return gmi.AcceptCIE
	}
	return gmi.None
}
