package grafanautils

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
)

func NewClient(url string, config gapi.Config) (*gapi.Client, error) {
	gapiClient, err := gapi.New(url, config)
	if err != nil {
		log.Infof("Error creating Grafana client for (%s): %v\n", url, err)
		return nil, err
	}
	return gapiClient, nil
}

func NewDataSourceQuery(from time.Time, to time.Time) gapi.DataSourceQuery {
	fromRFC3339 := from.Format(time.RFC3339)
	toRFC3339 := to.Format(time.RFC3339)
	fromUnixMilli := strconv.FormatInt(from.UnixMilli(), 10)
	toUnixMilli := strconv.FormatInt(to.UnixMilli(), 10)
	return gapi.DataSourceQuery{
		Queries: []gapi.DataSourceQueryQuery{},
		Range: gapi.QueryTimeRange{
			From: fromRFC3339,
			To:   toRFC3339,
			Raw: &gapi.DashboardRelativeTimeRange{
				From: "",
				To:   "",
			}},
		From: fromUnixMilli,
		To:   toUnixMilli,
	}
}

func NewDataSourceQueries(panels []gapi.DashboardPanel, from time.Time, to time.Time) []gapi.DataSourceQuery {
	var dataSourceQueries []gapi.DataSourceQuery
	for i, panel := range panels {
		dsq := NewDataSourceQuery(from, to)
		for _, target := range panel.Targets {
			dsqq := gapi.DataSourceQueryQuery{
				Expression: target.Expr,
				DataSource: gapi.DataSource{
					UID:  "prometheus",
					Type: "prometheus",
				},
				Interval:      target.Interval,
				LegendFormat:  target.LegendFormat,
				RefID:         target.RefID,
				RequestID:     strconv.Itoa(i) + target.RefID,
				QueryType:     "timeSeriesQuery",
				Exemplar:      false,
				UTCOffsetSec:  0,
				DatasourceID:  0,
				IntervalMs:    60 * time.Second.Milliseconds(),
				MaxDataPoints: 1000,
			}
			dsq.Queries = append(dsq.Queries, dsqq)
		}
		dataSourceQueries = append(dataSourceQueries, dsq)
	}
	return dataSourceQueries
}

func InjectPanelData(c *gapi.Client, dashboardModel *gapi.DashboardModel, queries []gapi.DataSourceQuery) error {
	for pidx := range (*dashboardModel).Panels {
		dsqResults, err := c.QueryDataSource(queries[pidx])
		if err != nil {
			log.Infof("Error retrieving Grafana DataSource Query results for Query (%v): %v\n", queries[pidx], err)
			continue
		}
		for _, result := range *dsqResults {
			for _, frame := range result.Frames {
				snapshotDataFields := []gapi.SnapshotField{}

				snapshotField1 := c.SchemaFieldToSnapshotField(frame.Schema.Fields[0])
				snapshotField2 := c.SchemaFieldToSnapshotField(frame.Schema.Fields[1])

				snapshotField1.Values = frame.Data.Values[0]
				snapshotField2.Values = frame.Data.Values[1]
				snapshotDataFields = append(snapshotDataFields, snapshotField1, snapshotField2)
				var metaMap map[string]interface{}
				metaJSON, err := json.Marshal(frame.Schema.Meta)
				if err != nil {
					log.Infof("Error marshalling frame.Schema.Meta for Frame (%v): %v\n", frame, err)
					continue
				}
				err = json.Unmarshal(metaJSON, &metaMap)
				if err != nil {
					log.Infof("Error unmarshalling frame.Schema.Meta for Frame (%v): %v\n", frame, err)
					continue
				}

				dashboardModel.Panels[pidx].SnapshotData = append(dashboardModel.Panels[pidx].SnapshotData, gapi.SnapshotData{
					Fields: snapshotDataFields,
					Meta:   metaMap,
					Name:   frame.Schema.Name,
					RefID:  frame.Schema.RefID,
				})
			}
		}
	}
	return nil
}

func GetDashboardSnapshot(c *gapi.Client, from time.Time, to time.Time, uid string, expires int64, external bool) (gapi.SnapshotCreateResponse, error) {
	s := gapi.Snapshot{}
	var res gapi.SnapshotCreateResponse

	dashboard, err := c.DashboardByUID(uid)
	if err != nil {
		log.Infof("Error getting Grafana dashboard with uid (%s): %v\n", uid, err)
		return res, err
	}
	// Set proper timerange for Snapshot
	fromRFC3339 := from.Format(time.RFC3339)
	toRFC3339 := to.Format(time.RFC3339)
	dashboard.Model.Time = gapi.QueryTimeRange{
		From: fromRFC3339,
		To:   toRFC3339,
		Raw: &gapi.DashboardRelativeTimeRange{
			From: "",
			To:   "",
		},
	}
	dataSourceQueries := NewDataSourceQueries(dashboard.Model.Panels, from, to)
	err = InjectPanelData(c, &dashboard.Model, dataSourceQueries)
	if err != nil {
		log.Infof("Error injecting panel data into dashboard (%v) with datasource queries (%v) from %v to %v: %v\n", dashboard, dataSourceQueries, from, to, err)
		return res, err
	}
	s.DashboardModel = dashboard.Model

	if expires > 0 {
		s.Expires = expires
	}
	s.External = external
	resp, err := c.NewSnapshot(s)

	if err != nil {
		log.Infof("Error creating Grafana Snapshot for Dashboard with uid (%s) from %v to %v: %v\n", uid, fromRFC3339, toRFC3339, err)
		return res, err
	}
	if resp == nil {
		return res, fmt.Errorf("failed to retrieve SnapshotResponse for Dashboard with UID (%v)", s.DashboardModel.UID)
	}
	res = *resp
	return res, nil
}

func GetDataSourceQueryResults(c *gapi.Client, dsqq gapi.DataSourceQueryQuery, from time.Time, to time.Time) (*gapi.QueryResults, gapi.DataSourceQuery, error) {
	dsq := NewDataSourceQuery(from, to)
	dsq.Queries = append(dsq.Queries, dsqq)
	dsqResult, err := c.QueryDataSource(dsq)
	if err != nil {
		log.Infof("Error retrieving Grafana DataSource Query results for Query (%v): %v\n", dsqq, err)
		return nil, dsq, err
	}
	return dsqResult, dsq, nil
}

func GetQueryValue(p promv1.API, q string, start time.Time, end time.Time) (model.Value, promv1.Warnings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r := promv1.Range{
		Start: start,
		End:   end,
		Step:  time.Minute,
	}
	result, warnings, err := p.QueryRange(ctx, q, r, promv1.WithTimeout(5*time.Second))
	if err != nil {
		log.Infof("Error querying Prometheus for (%s): %v\n", q, err)
		return nil, warnings, err
	}
	if len(warnings) > 0 {
		log.Infof("Warnings querying Prometheus for (%s): %v\n", q, warnings)
	}
	return result, warnings, nil
}
