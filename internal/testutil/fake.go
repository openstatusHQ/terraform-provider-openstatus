package testutil

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	monitorv1connect "buf.build/gen/go/openstatus/api/connectrpc/go/openstatus/monitor/v1/monitorv1connect"
	notificationv1connect "buf.build/gen/go/openstatus/api/connectrpc/go/openstatus/notification/v1/notificationv1connect"
	privatelocationv1connect "buf.build/gen/go/openstatus/api/connectrpc/go/openstatus/private_location/v1/private_locationv1connect"
	statuspagev1connect "buf.build/gen/go/openstatus/api/connectrpc/go/openstatus/status_page/v1/status_pagev1connect"
	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	notificationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/notification/v1"
	privatelocationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/private_location/v1"
	statuspagev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_page/v1"

	"connectrpc.com/connect"
)

const fixedTimestamp = "2026-01-01T00:00:00Z"

// Fake is an in-memory implementation of the OpenStatus services, faithful
// enough to drive a full Terraform lifecycle against.
type Fake struct {
	mu  sync.Mutex
	seq int

	httpMonitors     map[string]*monitorv1.HTTPMonitor
	tcpMonitors      map[string]*monitorv1.TCPMonitor
	dnsMonitors      map[string]*monitorv1.DNSMonitor
	notifications    map[string]*notificationv1.Notification
	statusPages      map[string]*statuspagev1.StatusPage
	components       map[string]*statuspagev1.PageComponent
	groups           map[string]*statuspagev1.PageComponentGroup
	privateLocations map[string]*privatelocationv1.PrivateLocation
}

func NewFake() *Fake {
	return &Fake{
		httpMonitors:     map[string]*monitorv1.HTTPMonitor{},
		tcpMonitors:      map[string]*monitorv1.TCPMonitor{},
		dnsMonitors:      map[string]*monitorv1.DNSMonitor{},
		notifications:    map[string]*notificationv1.Notification{},
		statusPages:      map[string]*statuspagev1.StatusPage{},
		components:       map[string]*statuspagev1.PageComponent{},
		groups:           map[string]*statuspagev1.PageComponentGroup{},
		privateLocations: map[string]*privatelocationv1.PrivateLocation{},
	}
}

func (f *Fake) nextID(prefix string) string {
	f.seq++
	return fmt.Sprintf("%s_%d", prefix, f.seq)
}

// Component returns the stored component, or nil when it does not exist. Tests
// use it to assert what actually reached the server, not just what the
// provider wrote to state.
func (f *Fake) Component(id string) *statuspagev1.PageComponent {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.components[id]
}

func notFound(kind, id string) *connect.Error {
	return connect.NewError(connect.CodeNotFound, fmt.Errorf("%s %q not found", kind, id))
}

type monitorService struct {
	monitorv1connect.UnimplementedMonitorServiceHandler
	f *Fake
}

func (s *monitorService) CreateHTTPMonitor(_ context.Context, req *connect.Request[monitorv1.CreateHTTPMonitorRequest]) (*connect.Response[monitorv1.CreateHTTPMonitorResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	m := req.Msg.GetMonitor()
	if m == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("monitor is required"))
	}
	m.SetId(s.f.nextID("http"))
	s.f.httpMonitors[m.GetId()] = m

	resp := &monitorv1.CreateHTTPMonitorResponse{}
	resp.SetMonitor(m)
	return connect.NewResponse(resp), nil
}

func (s *monitorService) CreateTCPMonitor(_ context.Context, req *connect.Request[monitorv1.CreateTCPMonitorRequest]) (*connect.Response[monitorv1.CreateTCPMonitorResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	m := req.Msg.GetMonitor()
	if m == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("monitor is required"))
	}
	m.SetId(s.f.nextID("tcp"))
	s.f.tcpMonitors[m.GetId()] = m

	resp := &monitorv1.CreateTCPMonitorResponse{}
	resp.SetMonitor(m)
	return connect.NewResponse(resp), nil
}

func (s *monitorService) CreateDNSMonitor(_ context.Context, req *connect.Request[monitorv1.CreateDNSMonitorRequest]) (*connect.Response[monitorv1.CreateDNSMonitorResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	m := req.Msg.GetMonitor()
	if m == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("monitor is required"))
	}
	m.SetId(s.f.nextID("dns"))
	s.f.dnsMonitors[m.GetId()] = m

	resp := &monitorv1.CreateDNSMonitorResponse{}
	resp.SetMonitor(m)
	return connect.NewResponse(resp), nil
}

func (s *monitorService) UpdateHTTPMonitor(_ context.Context, req *connect.Request[monitorv1.UpdateHTTPMonitorRequest]) (*connect.Response[monitorv1.UpdateHTTPMonitorResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	id := req.Msg.GetId()
	if _, ok := s.f.httpMonitors[id]; !ok {
		return nil, notFound("http monitor", id)
	}
	m := req.Msg.GetMonitor()
	m.SetId(id)
	s.f.httpMonitors[id] = m

	resp := &monitorv1.UpdateHTTPMonitorResponse{}
	resp.SetMonitor(m)
	return connect.NewResponse(resp), nil
}

func (s *monitorService) UpdateTCPMonitor(_ context.Context, req *connect.Request[monitorv1.UpdateTCPMonitorRequest]) (*connect.Response[monitorv1.UpdateTCPMonitorResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	id := req.Msg.GetId()
	if _, ok := s.f.tcpMonitors[id]; !ok {
		return nil, notFound("tcp monitor", id)
	}
	m := req.Msg.GetMonitor()
	m.SetId(id)
	s.f.tcpMonitors[id] = m

	resp := &monitorv1.UpdateTCPMonitorResponse{}
	resp.SetMonitor(m)
	return connect.NewResponse(resp), nil
}

func (s *monitorService) UpdateDNSMonitor(_ context.Context, req *connect.Request[monitorv1.UpdateDNSMonitorRequest]) (*connect.Response[monitorv1.UpdateDNSMonitorResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	id := req.Msg.GetId()
	if _, ok := s.f.dnsMonitors[id]; !ok {
		return nil, notFound("dns monitor", id)
	}
	m := req.Msg.GetMonitor()
	m.SetId(id)
	s.f.dnsMonitors[id] = m

	resp := &monitorv1.UpdateDNSMonitorResponse{}
	resp.SetMonitor(m)
	return connect.NewResponse(resp), nil
}

func (s *monitorService) GetMonitor(_ context.Context, req *connect.Request[monitorv1.GetMonitorRequest]) (*connect.Response[monitorv1.GetMonitorResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	id := req.Msg.GetId()
	config := &monitorv1.MonitorConfig{}
	switch {
	case s.f.httpMonitors[id] != nil:
		config.SetHttp(s.f.httpMonitors[id])
	case s.f.tcpMonitors[id] != nil:
		config.SetTcp(s.f.tcpMonitors[id])
	case s.f.dnsMonitors[id] != nil:
		config.SetDns(s.f.dnsMonitors[id])
	default:
		return nil, notFound("monitor", id)
	}

	resp := &monitorv1.GetMonitorResponse{}
	resp.SetMonitor(config)
	return connect.NewResponse(resp), nil
}

func (s *monitorService) ListMonitors(_ context.Context, _ *connect.Request[monitorv1.ListMonitorsRequest]) (*connect.Response[monitorv1.ListMonitorsResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	https := make([]*monitorv1.HTTPMonitor, 0, len(s.f.httpMonitors))
	for _, m := range s.f.httpMonitors {
		https = append(https, m)
	}
	sort.Slice(https, func(i, j int) bool { return https[i].GetId() < https[j].GetId() })

	tcps := make([]*monitorv1.TCPMonitor, 0, len(s.f.tcpMonitors))
	for _, m := range s.f.tcpMonitors {
		tcps = append(tcps, m)
	}
	sort.Slice(tcps, func(i, j int) bool { return tcps[i].GetId() < tcps[j].GetId() })

	dnss := make([]*monitorv1.DNSMonitor, 0, len(s.f.dnsMonitors))
	for _, m := range s.f.dnsMonitors {
		dnss = append(dnss, m)
	}
	sort.Slice(dnss, func(i, j int) bool { return dnss[i].GetId() < dnss[j].GetId() })

	resp := &monitorv1.ListMonitorsResponse{}
	resp.SetHttpMonitors(https)
	resp.SetTcpMonitors(tcps)
	resp.SetDnsMonitors(dnss)
	resp.SetTotalSize(int32(len(https) + len(tcps) + len(dnss)))
	return connect.NewResponse(resp), nil
}

func (s *monitorService) DeleteMonitor(_ context.Context, req *connect.Request[monitorv1.DeleteMonitorRequest]) (*connect.Response[monitorv1.DeleteMonitorResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	id := req.Msg.GetId()
	_, isHTTP := s.f.httpMonitors[id]
	_, isTCP := s.f.tcpMonitors[id]
	_, isDNS := s.f.dnsMonitors[id]
	if !isHTTP && !isTCP && !isDNS {
		return nil, notFound("monitor", id)
	}
	delete(s.f.httpMonitors, id)
	delete(s.f.tcpMonitors, id)
	delete(s.f.dnsMonitors, id)

	resp := &monitorv1.DeleteMonitorResponse{}
	resp.SetSuccess(true)
	return connect.NewResponse(resp), nil
}

type notificationService struct {
	notificationv1connect.UnimplementedNotificationServiceHandler
	f *Fake
}

// stripNtfyToken mirrors production, which never echoes ntfy.token back.
func stripNtfyToken(n *notificationv1.Notification) *notificationv1.Notification {
	data := n.GetData()
	if data == nil || data.GetNtfy() == nil {
		return n
	}
	clone := &notificationv1.Notification{}
	clone.SetId(n.GetId())
	clone.SetName(n.GetName())
	clone.SetProvider(n.GetProvider())
	clone.SetMonitorIds(n.GetMonitorIds())
	clone.SetCreatedAt(n.GetCreatedAt())
	clone.SetUpdatedAt(n.GetUpdatedAt())

	ntfy := &notificationv1.NtfyData{}
	ntfy.SetTopic(data.GetNtfy().GetTopic())
	ntfy.SetServerUrl(data.GetNtfy().GetServerUrl())
	cloneData := &notificationv1.NotificationData{}
	cloneData.SetNtfy(ntfy)
	clone.SetData(cloneData)
	return clone
}

func (s *notificationService) CreateNotification(_ context.Context, req *connect.Request[notificationv1.CreateNotificationRequest]) (*connect.Response[notificationv1.CreateNotificationResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	n := &notificationv1.Notification{}
	n.SetId(s.f.nextID("notif"))
	n.SetName(req.Msg.GetName())
	n.SetProvider(req.Msg.GetProvider())
	n.SetData(req.Msg.GetData())
	n.SetMonitorIds(req.Msg.GetMonitorIds())
	n.SetCreatedAt(fixedTimestamp)
	n.SetUpdatedAt(fixedTimestamp)
	s.f.notifications[n.GetId()] = n

	resp := &notificationv1.CreateNotificationResponse{}
	resp.SetNotification(stripNtfyToken(n))
	return connect.NewResponse(resp), nil
}

func (s *notificationService) GetNotification(_ context.Context, req *connect.Request[notificationv1.GetNotificationRequest]) (*connect.Response[notificationv1.GetNotificationResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	n, ok := s.f.notifications[req.Msg.GetId()]
	if !ok {
		return nil, notFound("notification", req.Msg.GetId())
	}

	resp := &notificationv1.GetNotificationResponse{}
	resp.SetNotification(stripNtfyToken(n))
	return connect.NewResponse(resp), nil
}

func (s *notificationService) UpdateNotification(_ context.Context, req *connect.Request[notificationv1.UpdateNotificationRequest]) (*connect.Response[notificationv1.UpdateNotificationResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	n, ok := s.f.notifications[req.Msg.GetId()]
	if !ok {
		return nil, notFound("notification", req.Msg.GetId())
	}
	if req.Msg.HasName() {
		n.SetName(req.Msg.GetName())
	}
	if req.Msg.GetData() != nil {
		n.SetData(req.Msg.GetData())
	}
	if req.Msg.GetUpdateMonitorIds() {
		n.SetMonitorIds(req.Msg.GetMonitorIds())
	}

	resp := &notificationv1.UpdateNotificationResponse{}
	resp.SetNotification(stripNtfyToken(n))
	return connect.NewResponse(resp), nil
}

func (s *notificationService) DeleteNotification(_ context.Context, req *connect.Request[notificationv1.DeleteNotificationRequest]) (*connect.Response[notificationv1.DeleteNotificationResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	if _, ok := s.f.notifications[req.Msg.GetId()]; !ok {
		return nil, notFound("notification", req.Msg.GetId())
	}
	delete(s.f.notifications, req.Msg.GetId())

	resp := &notificationv1.DeleteNotificationResponse{}
	resp.SetSuccess(true)
	return connect.NewResponse(resp), nil
}

type statusPageService struct {
	statuspagev1connect.UnimplementedStatusPageServiceHandler
	f *Fake
}

func (s *statusPageService) CreateStatusPage(_ context.Context, req *connect.Request[statuspagev1.CreateStatusPageRequest]) (*connect.Response[statuspagev1.CreateStatusPageResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	m := req.Msg
	p := &statuspagev1.StatusPage{}
	p.SetId(s.f.nextID("page"))
	p.SetTitle(m.GetTitle())
	p.SetSlug(m.GetSlug())
	p.SetDescription(m.GetDescription())
	p.SetHomepageUrl(m.GetHomepageUrl())
	p.SetContactUrl(m.GetContactUrl())
	p.SetIcon(m.GetIcon())
	p.SetCustomDomain(m.GetCustomDomain())
	p.SetTheme(m.GetTheme())
	accessType := m.GetAccessType()
	if accessType == statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_UNSPECIFIED {
		accessType = statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_PUBLIC
	}
	p.SetAccessType(accessType)
	p.SetPassword(m.GetPassword())
	p.SetAuthEmailDomains(m.GetAuthEmailDomains())
	p.SetAllowedIpRanges(m.GetAllowedIpRanges())
	p.SetDefaultLocale(m.GetDefaultLocale())
	p.SetLocales(m.GetLocales())
	p.SetAllowIndex(m.GetAllowIndex())
	p.SetCustomTheme(m.GetCustomTheme())
	p.SetCreatedAt(fixedTimestamp)
	p.SetUpdatedAt(fixedTimestamp)
	s.f.statusPages[p.GetId()] = p

	resp := &statuspagev1.CreateStatusPageResponse{}
	resp.SetStatusPage(p)
	return connect.NewResponse(resp), nil
}

func (s *statusPageService) GetStatusPage(_ context.Context, req *connect.Request[statuspagev1.GetStatusPageRequest]) (*connect.Response[statuspagev1.GetStatusPageResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	p, ok := s.f.statusPages[req.Msg.GetId()]
	if !ok {
		return nil, notFound("status page", req.Msg.GetId())
	}

	resp := &statuspagev1.GetStatusPageResponse{}
	resp.SetStatusPage(p)
	return connect.NewResponse(resp), nil
}

func (s *statusPageService) UpdateStatusPage(_ context.Context, req *connect.Request[statuspagev1.UpdateStatusPageRequest]) (*connect.Response[statuspagev1.UpdateStatusPageResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	m := req.Msg
	p, ok := s.f.statusPages[m.GetId()]
	if !ok {
		return nil, notFound("status page", m.GetId())
	}
	if m.HasTitle() {
		p.SetTitle(m.GetTitle())
	}
	if m.HasSlug() {
		p.SetSlug(m.GetSlug())
	}
	if m.HasDescription() {
		p.SetDescription(m.GetDescription())
	}
	if m.HasHomepageUrl() {
		p.SetHomepageUrl(m.GetHomepageUrl())
	}
	if m.HasContactUrl() {
		p.SetContactUrl(m.GetContactUrl())
	}
	if m.HasIcon() {
		p.SetIcon(m.GetIcon())
	}
	if m.HasCustomDomain() {
		p.SetCustomDomain(m.GetCustomDomain())
	}
	if m.HasTheme() {
		p.SetTheme(m.GetTheme())
	}
	if m.HasAccessType() {
		p.SetAccessType(m.GetAccessType())
	}
	if m.HasPassword() {
		p.SetPassword(m.GetPassword())
	}
	if m.HasAllowedIpRanges() {
		p.SetAllowedIpRanges(m.GetAllowedIpRanges())
	}
	if m.HasDefaultLocale() {
		p.SetDefaultLocale(m.GetDefaultLocale())
	}
	if m.HasAllowIndex() {
		p.SetAllowIndex(m.GetAllowIndex())
	}
	p.SetAuthEmailDomains(m.GetAuthEmailDomains())
	p.SetLocales(m.GetLocales())
	p.SetCustomTheme(m.GetCustomTheme())

	resp := &statuspagev1.UpdateStatusPageResponse{}
	resp.SetStatusPage(p)
	return connect.NewResponse(resp), nil
}

func (s *statusPageService) DeleteStatusPage(_ context.Context, req *connect.Request[statuspagev1.DeleteStatusPageRequest]) (*connect.Response[statuspagev1.DeleteStatusPageResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	if _, ok := s.f.statusPages[req.Msg.GetId()]; !ok {
		return nil, notFound("status page", req.Msg.GetId())
	}
	delete(s.f.statusPages, req.Msg.GetId())

	resp := &statuspagev1.DeleteStatusPageResponse{}
	resp.SetSuccess(true)
	return connect.NewResponse(resp), nil
}

func (s *statusPageService) AddMonitorComponent(_ context.Context, req *connect.Request[statuspagev1.AddMonitorComponentRequest]) (*connect.Response[statuspagev1.AddMonitorComponentResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	m := req.Msg
	c := &statuspagev1.PageComponent{}
	c.SetId(s.f.nextID("comp"))
	c.SetPageId(m.GetPageId())
	c.SetMonitorId(m.GetMonitorId())
	c.SetName(m.GetName())
	c.SetDescription(m.GetDescription())
	c.SetOrder(m.GetOrder())
	c.SetGroupId(m.GetGroupId())
	c.SetType(statuspagev1.PageComponentType_PAGE_COMPONENT_TYPE_MONITOR)
	c.SetCreatedAt(fixedTimestamp)
	c.SetUpdatedAt(fixedTimestamp)
	s.f.components[c.GetId()] = c

	resp := &statuspagev1.AddMonitorComponentResponse{}
	resp.SetComponent(c)
	return connect.NewResponse(resp), nil
}

func (s *statusPageService) AddStaticComponent(_ context.Context, req *connect.Request[statuspagev1.AddStaticComponentRequest]) (*connect.Response[statuspagev1.AddStaticComponentResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	m := req.Msg
	c := &statuspagev1.PageComponent{}
	c.SetId(s.f.nextID("comp"))
	c.SetPageId(m.GetPageId())
	c.SetName(m.GetName())
	c.SetDescription(m.GetDescription())
	c.SetOrder(m.GetOrder())
	c.SetGroupId(m.GetGroupId())
	c.SetType(statuspagev1.PageComponentType_PAGE_COMPONENT_TYPE_STATIC)
	c.SetCreatedAt(fixedTimestamp)
	c.SetUpdatedAt(fixedTimestamp)
	s.f.components[c.GetId()] = c

	resp := &statuspagev1.AddStaticComponentResponse{}
	resp.SetComponent(c)
	return connect.NewResponse(resp), nil
}

func (s *statusPageService) UpdateComponent(_ context.Context, req *connect.Request[statuspagev1.UpdateComponentRequest]) (*connect.Response[statuspagev1.UpdateComponentResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	m := req.Msg
	c, ok := s.f.components[m.GetId()]
	if !ok {
		return nil, notFound("component", m.GetId())
	}
	if m.HasName() {
		c.SetName(m.GetName())
	}
	if m.HasDescription() {
		c.SetDescription(m.GetDescription())
	}
	if m.HasOrder() {
		c.SetOrder(m.GetOrder())
	}
	if m.HasGroupId() {
		c.SetGroupId(m.GetGroupId())
	}
	if m.HasGroupOrder() {
		c.SetGroupOrder(m.GetGroupOrder())
	}

	resp := &statuspagev1.UpdateComponentResponse{}
	resp.SetComponent(c)
	return connect.NewResponse(resp), nil
}

func (s *statusPageService) RemoveComponent(_ context.Context, req *connect.Request[statuspagev1.RemoveComponentRequest]) (*connect.Response[statuspagev1.RemoveComponentResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	if _, ok := s.f.components[req.Msg.GetId()]; !ok {
		return nil, notFound("component", req.Msg.GetId())
	}
	delete(s.f.components, req.Msg.GetId())

	resp := &statuspagev1.RemoveComponentResponse{}
	resp.SetSuccess(true)
	return connect.NewResponse(resp), nil
}

func (s *statusPageService) CreateComponentGroup(_ context.Context, req *connect.Request[statuspagev1.CreateComponentGroupRequest]) (*connect.Response[statuspagev1.CreateComponentGroupResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	g := &statuspagev1.PageComponentGroup{}
	g.SetId(s.f.nextID("group"))
	g.SetPageId(req.Msg.GetPageId())
	g.SetName(req.Msg.GetName())
	g.SetDefaultOpen(req.Msg.GetDefaultOpen())
	g.SetCreatedAt(fixedTimestamp)
	g.SetUpdatedAt(fixedTimestamp)
	s.f.groups[g.GetId()] = g

	resp := &statuspagev1.CreateComponentGroupResponse{}
	resp.SetGroup(g)
	return connect.NewResponse(resp), nil
}

func (s *statusPageService) UpdateComponentGroup(_ context.Context, req *connect.Request[statuspagev1.UpdateComponentGroupRequest]) (*connect.Response[statuspagev1.UpdateComponentGroupResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	g, ok := s.f.groups[req.Msg.GetId()]
	if !ok {
		return nil, notFound("component group", req.Msg.GetId())
	}
	if req.Msg.HasName() {
		g.SetName(req.Msg.GetName())
	}
	if req.Msg.HasDefaultOpen() {
		g.SetDefaultOpen(req.Msg.GetDefaultOpen())
	}

	resp := &statuspagev1.UpdateComponentGroupResponse{}
	resp.SetGroup(g)
	return connect.NewResponse(resp), nil
}

func (s *statusPageService) DeleteComponentGroup(_ context.Context, req *connect.Request[statuspagev1.DeleteComponentGroupRequest]) (*connect.Response[statuspagev1.DeleteComponentGroupResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	if _, ok := s.f.groups[req.Msg.GetId()]; !ok {
		return nil, notFound("component group", req.Msg.GetId())
	}
	delete(s.f.groups, req.Msg.GetId())

	resp := &statuspagev1.DeleteComponentGroupResponse{}
	resp.SetSuccess(true)
	return connect.NewResponse(resp), nil
}

func (s *statusPageService) GetStatusPageContent(_ context.Context, req *connect.Request[statuspagev1.GetStatusPageContentRequest]) (*connect.Response[statuspagev1.GetStatusPageContentResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	id := req.Msg.GetId()
	p, ok := s.f.statusPages[id]
	if !ok {
		for _, candidate := range s.f.statusPages {
			if candidate.GetSlug() == req.Msg.GetSlug() {
				p, ok = candidate, true
				break
			}
		}
	}
	if !ok {
		return nil, notFound("status page", id)
	}

	components := make([]*statuspagev1.PageComponent, 0, len(s.f.components))
	for _, c := range s.f.components {
		if c.GetPageId() == p.GetId() {
			components = append(components, c)
		}
	}
	sort.Slice(components, func(i, j int) bool { return components[i].GetId() < components[j].GetId() })

	groups := make([]*statuspagev1.PageComponentGroup, 0, len(s.f.groups))
	for _, g := range s.f.groups {
		if g.GetPageId() == p.GetId() {
			groups = append(groups, g)
		}
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].GetId() < groups[j].GetId() })

	resp := &statuspagev1.GetStatusPageContentResponse{}
	resp.SetStatusPage(p)
	resp.SetComponents(components)
	resp.SetGroups(groups)
	return connect.NewResponse(resp), nil
}

type privateLocationService struct {
	privatelocationv1connect.UnimplementedPrivateLocationServiceHandler
	f *Fake
}

func (s *privateLocationService) CreatePrivateLocation(_ context.Context, req *connect.Request[privatelocationv1.CreatePrivateLocationRequest]) (*connect.Response[privatelocationv1.CreatePrivateLocationResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	l := &privatelocationv1.PrivateLocation{}
	l.SetId(s.f.nextID("pl"))
	l.SetName(req.Msg.GetName())
	l.SetToken("tk_" + l.GetId())
	l.SetMonitorIds(req.Msg.GetMonitorIds())
	l.SetMetadata(req.Msg.GetMetadata())
	l.SetStatus(privatelocationv1.PrivateLocationStatus_PRIVATE_LOCATION_STATUS_ACTIVE)
	l.SetCreatedAt(fixedTimestamp)
	l.SetUpdatedAt(fixedTimestamp)
	s.f.privateLocations[l.GetId()] = l

	resp := &privatelocationv1.CreatePrivateLocationResponse{}
	resp.SetPrivateLocation(l)
	return connect.NewResponse(resp), nil
}

func (s *privateLocationService) GetPrivateLocation(_ context.Context, req *connect.Request[privatelocationv1.GetPrivateLocationRequest]) (*connect.Response[privatelocationv1.GetPrivateLocationResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	l, ok := s.f.privateLocations[req.Msg.GetId()]
	if !ok {
		return nil, notFound("private location", req.Msg.GetId())
	}

	resp := &privatelocationv1.GetPrivateLocationResponse{}
	resp.SetPrivateLocation(l)
	return connect.NewResponse(resp), nil
}

func (s *privateLocationService) ListPrivateLocations(_ context.Context, _ *connect.Request[privatelocationv1.ListPrivateLocationsRequest]) (*connect.Response[privatelocationv1.ListPrivateLocationsResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	summaries := make([]*privatelocationv1.PrivateLocationSummary, 0, len(s.f.privateLocations))
	for _, l := range s.f.privateLocations {
		// The list endpoint deliberately omits the agent token.
		summary := &privatelocationv1.PrivateLocationSummary{}
		summary.SetId(l.GetId())
		summary.SetName(l.GetName())
		summary.SetMonitorCount(int32(len(l.GetMonitorIds())))
		summary.SetMetadata(l.GetMetadata())
		summary.SetStatus(l.GetStatus())
		summary.SetLastSeenAt(l.GetLastSeenAt())
		summary.SetCreatedAt(l.GetCreatedAt())
		summary.SetUpdatedAt(l.GetUpdatedAt())
		summaries = append(summaries, summary)
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].GetId() < summaries[j].GetId() })

	resp := &privatelocationv1.ListPrivateLocationsResponse{}
	resp.SetPrivateLocations(summaries)
	resp.SetTotalSize(int32(len(summaries)))
	return connect.NewResponse(resp), nil
}

func (s *privateLocationService) UpdatePrivateLocation(_ context.Context, req *connect.Request[privatelocationv1.UpdatePrivateLocationRequest]) (*connect.Response[privatelocationv1.UpdatePrivateLocationResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	l, ok := s.f.privateLocations[req.Msg.GetId()]
	if !ok {
		return nil, notFound("private location", req.Msg.GetId())
	}
	if req.Msg.HasName() {
		l.SetName(req.Msg.GetName())
	}
	// monitor_ids and metadata only apply when their sentinel is set.
	if req.Msg.GetUpdateMonitorIds() {
		l.SetMonitorIds(req.Msg.GetMonitorIds())
	}
	if req.Msg.GetUpdateMetadata() {
		l.SetMetadata(req.Msg.GetMetadata())
	}

	resp := &privatelocationv1.UpdatePrivateLocationResponse{}
	resp.SetPrivateLocation(l)
	return connect.NewResponse(resp), nil
}

func (s *privateLocationService) DeletePrivateLocation(_ context.Context, req *connect.Request[privatelocationv1.DeletePrivateLocationRequest]) (*connect.Response[privatelocationv1.DeletePrivateLocationResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	if _, ok := s.f.privateLocations[req.Msg.GetId()]; !ok {
		return nil, notFound("private location", req.Msg.GetId())
	}
	delete(s.f.privateLocations, req.Msg.GetId())

	resp := &privatelocationv1.DeletePrivateLocationResponse{}
	resp.SetSuccess(true)
	return connect.NewResponse(resp), nil
}
