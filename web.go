package main

import (
	"bytes"
	"html/template"
	"net/http"
	"sort"
	"time"

	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/image-migration-dashboard/internal/core"
	"github.com/wcharczuk/go-chart"
)

var homePageTemplate = template.Must(template.New("homepage").Parse(`
<!doctype html>
<html class="no-js" lang="en">

<head>
	<meta charset="utf-8">
	<title>Image Migration Dashboard</title>
	<meta name="description" content="Dashboard for the ongoing migration from Quay to Keppel.">
	<meta name="viewport" content="width=device-width, initial-scale=1">

	<link rel="stylesheet" type="text/css" href="//fonts.googleapis.com/css?family=Raleway:400,300,600">
	<link rel="stylesheet" href="//cdnjs.cloudflare.com/ajax/libs/normalize/8.0.1/normalize.min.css">
	<link rel="stylesheet" href="//cdnjs.cloudflare.com/ajax/libs/skeleton/2.0.4/skeleton.min.css">
	<style type="text/css">
		section.header {
			margin-top: 0.5em;
		}

		div.container.wide {
			width: 95%;
			max-width: initial;
		}

		ul > li {
			margin-bottom: 0.25em;
			line-height: 1.2;
		}
	</style>
</head>

<body>
	<div class="container">
		<section class="header">
			<h2 class="title" style="text-align: center;">Image Migration Dashboard</h2>
		</section>
	</div>

	<div class="container wide">
		<div class="row">
			<!-- Image usage over time container -->
			<div class="nine columns">
				<h4>Image sources over time</h4>
				<img class="u-max-full-width" src="/graph.png">
			</div>

			<!-- Image distribution container -->
			<div class="three columns">
				<h4>As of today</h4>
				<img class="u-max-full-width" src="/donut.png">
				<p>
					{{- $n := .LastResult.NoOfImages -}}
					{{$n.Quay}} Quay + {{$n.Keppel}} Keppel + {{$n.Misc}} Misc = {{$n.Total -}}
				</p>
			</div>
		</div>
	</div>

	<!-- Images container -->
	<div class="container">
		<hr>
		{{ range $reg := .Registries }}
		<h4>Images currently coming from {{ $reg.Name }}</h4>
		<table class="u-full-width">
			<thead>
				<tr>
					<th style="max-width: 350px;;">Image</th>
					<th>Namespace/Pod/Container</th>
				</tr>
			</thead>
			<tbody>
				{{ range $img := $reg.Images }}
				<tr>
					<td style="max-width: 350px;; word-wrap: break-word;">{{ $img.Name }}</td>
					<td>
						<ul>
						{{ range $v := $img.Containers }}
							<li>
								{{ $v }}
							</li>
						{{ end }}
						</ul>
					</td>
				</tr>
				{{ end }}
			</tbody>
		</table>
		{{ end }}

	</div>
</body>

</html>
`))

///////////////////////////////////////////////////////////////////////////////
// http.HandleFunc(s)

func handleHomePage(w http.ResponseWriter, r *http.Request) {
	db.RW.RLock()
	res := db.DailyResults[db.LastScrapeTime.Format(core.ISODateFormat)]
	images := db.Images
	db.RW.RUnlock()

	// sort everything alphabetically
	var data struct {
		LastResult core.ScanResult
		Registries []struct {
			Name   string
			Images []core.Image
		}
	}
	data.LastResult = res
	data.Registries = append(data.Registries, []struct {
		Name   string
		Images []core.Image
	}{
		{"Quay", images.Quay},
		{"Keppel", images.Keppel},
		{"Misc.", images.Misc},
	}...)

	homePageTemplate.Execute(w, data)
}

// HandleGetDonutChart serves donuts.
func handleGetDonutChart(w http.ResponseWriter, r *http.Request) {
	db.RW.RLock()
	res := db.DailyResults[db.LastScrapeTime.Format(core.ISODateFormat)]
	db.RW.RUnlock()

	donut := chart.DonutChart{
		Width:  512,
		Height: 512,
		Values: []chart.Value{
			{Value: float64(res.NoOfImages.Keppel), Label: "Keppel"},
			{Value: float64(res.NoOfImages.Quay), Label: "Quay"},
			{Value: float64(res.NoOfImages.Misc), Label: "Misc."},
		},
	}
	var b bytes.Buffer
	err := donut.Render(chart.PNG, &b)
	if err != nil {
		logg.Error(err.Error())
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(b.Bytes())
}

// HandleGetGraph serves the graph.
func handleGetGraph(w http.ResponseWriter, r *http.Request) {
	db.RW.RLock()
	ts := []time.Time{}
	kfs := []float64{}
	qfs := []float64{}

	//if we just do `range db.DailyResults`, we get a sort-of-random order
	//because it's a map; but we need the correct time order to render the graphs
	//correctly
	dateStrings := []string{}
	for k := range db.DailyResults {
		dateStrings = append(dateStrings, k)
	}
	sort.Strings(dateStrings)
	for _, dateString := range dateStrings {
		v := db.DailyResults[dateString]
		ts = append(ts, time.Unix(v.ScrapedAt, 0))
		kfs = append(kfs, float64(v.NoOfImages.Keppel))
		qfs = append(qfs, float64(v.NoOfImages.Quay))
	}
	db.RW.RUnlock()

	graph := chart.Chart{
		Background: chart.Style{
			Padding: chart.Box{
				Top:    20,
				Left:   60,
				Right:  20,
				Bottom: 20,
			},
		},
		Series: []chart.Series{
			chart.TimeSeries{
				Name:    "Keppel",
				XValues: ts,
				YValues: kfs,
			},
			chart.TimeSeries{
				Name:    "Quay",
				XValues: ts,
				YValues: qfs,
			},
		},
	}
	//note we have to do this as a separate step because we need a reference to graph
	graph.Elements = []chart.Renderable{
		chart.LegendLeft(&graph),
	}
	var b bytes.Buffer
	err := graph.Render(chart.PNG, &b)
	if err != nil {
		logg.Error(err.Error())
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(b.Bytes())
}
