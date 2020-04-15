package main

import (
	"bytes"
	"html/template"
	"net/http"
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
</head>

<body>
	<!-- Primary page container -->
	<div class="container">
		<section class="header">
			<h2 class="title" style="text-align: center;">Image Migration Dashboard</h2>
		</section>

		<!-- Image usage over time container -->
		<div class="container">
			<div class="row">
				<div class="twelve columns">
					<h4>Image usage over time</h4>
				</div>
			</div>
			<div class="row">
				<div class="twelve columns">
					<img class="u-max-full-width" src="/graph.png">
				</div>
			</div>
		</div>

		<!-- Image distribution container -->
		<div class="container">
			<div class="row">
				<div class="twelve columns">
					<h4>Image distribution</h4>
				</div>
			</div>
			<div class="row">
				<div class="one-half column">
					<img class="u-max-full-width" src="/donut.png">
				</div>
				<div class="one-half column">
					<ul>
						<li>Total: {{ .LastResult.NoOfImages.Total }}</li>
						<li>Keppel: {{ .LastResult.NoOfImages.Keppel }}</li>
						<li>Quay: {{ .LastResult.NoOfImages.Quay }}</li>
						<li>Misc.: {{ .LastResult.NoOfImages.Misc }}</li>
					</ul>
				</div>
			</div>
		</div>

		<!-- Images container -->
		<div class="container">
			<div class="row">
				<div class="twelve columns">
					<h4>Images</h4>
				</div>
			</div>

			{{ range $reg, $imgs := .Images }}
			<div class="row">
				<div class="twelve columns">
					<h5>{{ $reg }}</h5>
				</div>
			</div>
			<div class="row">
				<div class="twelve columns">
					<table class="u-full-width">
						<thead>
							<tr>
								<th>Image</th>
								<th>Namespace/Pod/Container</th>
							</tr>
						</thead>
						<tbody>
							{{ range $name, $cntrs := $imgs }}
							<tr>
								<td>{{ $name }}</td>
								<td>
									<ul>
									{{ range $v := $cntrs }}
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
				</div>
			</div>
			{{ end }}

		</div>
	</div> <!-- Primary page container -->
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

	data := struct {
		LastResult core.ScanResult
		Images     core.ImageData
	}{res, images}
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
	for _, v := range db.DailyResults {
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
