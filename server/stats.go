package server

import (
	"context"
	"github.com/prometheus/common/log"
	"go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

type Stats struct {
	prometheusExporter *prometheus.Exporter
	mSocketRequest *stats.Int64Measure
	mSocketConnection *stats.Int64Measure
	mRequest *stats.Int64Measure
}

func NewStatsHolder() *Stats {

	mSocketRequest := stats.Int64("spaceship/socket_requests", "Socket Request Count", "By")
	vSocketRequestSum := &view.View{
		Name: "spaceship/socket_requests_sum",
		Measure: mSocketRequest,
		Description: "The number of total socket request",
		Aggregation: view.Sum(),
	}

	mSocketConnection := stats.Int64("spaceship/socket_connection", "Socket Counnection Count", "By")
	vSocketConnectionSum := &view.View{
		Name: "spaceship/socket_connection_sum",
		Measure: mSocketConnection,
		Description: "The number of total socket connection",
		Aggregation: view.Sum(),
	}

	mRequest := stats.Int64("spaceship/requests", "Request Count", "By")
	vRequestSum := &view.View{
		Name: "spaceship/requests_sum",
		Measure: mRequest,
		Description: "The number of total request",
		Aggregation: view.Sum(),
	}

	if err := view.Register(vSocketRequestSum, vSocketConnectionSum, vRequestSum); err != nil {
		log.Fatalln("Error while registering stat views")
	}

	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "spaceship",
	})
	if err != nil {
		log.Fatalln("Error while creating new prometheus exporter")
	}

	view.RegisterExporter(pe)

	return &Stats{
		prometheusExporter: pe,
		mSocketRequest: mSocketRequest,
		mSocketConnection: mSocketConnection,
		mRequest: mRequest,
	}

}

func (s Stats) IncrSocketRequest(){

	ctx, _ := tag.New(context.Background())
	stats.Record(ctx, s.mSocketRequest.M(1))

}

func (s Stats) IncrRequest(){

	ctx, _ := tag.New(context.Background())
	stats.Record(ctx, s.mRequest.M(1))

}

func (s Stats) IncrSocketConnection(){

	ctx, _ := tag.New(context.Background())
	stats.Record(ctx, s.mSocketConnection.M(1))

}

func (s Stats) DecrSocketConnection(){

	ctx, _ := tag.New(context.Background())
	stats.Record(ctx, s.mSocketConnection.M(-1))

}