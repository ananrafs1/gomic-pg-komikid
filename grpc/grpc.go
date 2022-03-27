package grpc

import (
	"github.com/ananrafs1/gomic-pg-komikid/komikid"
	"github.com/ananrafs1/gomic/model"
	"github.com/ananrafs1/gomic/orchestrator"
	pg "github.com/ananrafs1/gomic/orchestrator/shared/plugin"
	"github.com/hashicorp/go-plugin"
)

type Scrapper struct{}

func (Scrapper) Scrap(Title string, Page, Quantity int) model.Comic {
	return komikid.Extract(Title, Page, Quantity)
}

func (Scrapper) ScrapPerChapter(Title, Id string) model.Chapter {
	return model.Chapter{}
}

func Serve() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: orchestrator.Handshake,
		Plugins: map[string]plugin.Plugin{
			"grpcscrapper": &pg.GRPCPlugin{Impl: &Scrapper{}},
		},

		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
