package butler

import "time"

type UsageRecordStep struct {
	// one of: "auth", "middleware", "handler", "internal:etag", "internal:encoding"
	Step string
	// only for middleware step, name of the middleware
	Name  string
	Start *time.Time
	End   *time.Time
}

type UsageRecord struct {
	Method string
	// url path requested by the client (e.x. /api/book/123)
	UrlPath string
	// the pattern as provided to the Endpoint and/or the Groups  (e.x. /api/book/:id)
	PathPattern string
	Steps       []UsageRecordStep
	Start       *time.Time
	End         *time.Time
	// The HTTP status of the final response
	// unavailable for custom handlers
	RespStatus uint
	// Total length of the body
	//
	// For streamed responses this is the total length not the actually delivered amount
	RespLen uint
}

type UsageMonitor interface {
	Record(entry *UsageRecord)
}

type RecordBuilder interface {
	StepStart(s, name string)
	StepEnd(s, name string)
	GetRecord() *UsageRecord
}

type monitorRecorder interface {
	CreateRecord(method, routePattern, urlpath string) RecordBuilder
	FinalizeRecord(b RecordBuilder, status, respLen uint)
}

type voidRecorder struct{}

func (voidRecorder) CreateRecord(method, routePattern, urlpath string) RecordBuilder {
	return &voidRecord{}
}

func (voidRecorder) FinalizeRecord(rb RecordBuilder, status, respLen uint) {}

type voidRecord struct{}

func (voidRecord) StepStart(step, name string) {}

func (voidRecord) StepEnd(step, name string) {}

func (voidRecord) GetRecord() *UsageRecord {
	panic("void recorder does not create usage records")
}

type usageMonitorRecorder struct {
	usageMonitor UsageMonitor
}

func (usageMonitorRecorder) CreateRecord(method, routePattern, urlpath string) RecordBuilder {
	now := time.Now()
	rec := UsageRecord{
		Method:      method,
		PathPattern: routePattern,
		UrlPath:     urlpath,
		Steps:       []UsageRecordStep{},
		Start:       &now,
	}

	return &usageMonitorRecord{&rec}
}

func (umr *usageMonitorRecorder) FinalizeRecord(r RecordBuilder, status, respLen uint) {
	now := time.Now()
	rec := r.GetRecord()
	rec.End = &now
	rec.RespStatus = status
	rec.RespLen = respLen
	go umr.usageMonitor.Record(rec)
}

type usageMonitorRecord struct {
	record *UsageRecord
}

func (r *usageMonitorRecord) StepStart(step, name string) {
	now := time.Now()
	r.record.Steps = append(r.record.Steps, UsageRecordStep{
		Step:  step,
		Name:  name,
		Start: &now,
	})
}

func (r *usageMonitorRecord) StepEnd(step, name string) {
	for idx := range r.record.Steps {
		s := &r.record.Steps[idx]
		if s.Step == step && s.Name == name {
			now := time.Now()
			s.End = &now
			return
		}
	}
}

func (r *usageMonitorRecord) GetRecord() *UsageRecord {
	return r.record
}

func createMonitorRecorder(server *Server) monitorRecorder {
	monitor := server.usageMonitor

	if monitor == nil {
		return voidRecorder{}
	}

	return &usageMonitorRecorder{monitor}
}

type mstep struct {
	Auth          string
	ReqMiddleware string
	ResMiddleware string
	Handler       string
	EtagHandler   string
	Encoding      string
	Custom        string
	BindinParams  string
	Streaming     string
}

var MonitorStep = mstep{
	Auth:          "auth",
	ReqMiddleware: "middleware:request",
	ResMiddleware: "middleware:response",
	Handler:       "handler",
	EtagHandler:   "internal:etag",
	Encoding:      "internal:encoding",
	BindinParams:  "internal:bind_params",
	Streaming:     "internal:streaming",
	Custom:        "custom",
}
