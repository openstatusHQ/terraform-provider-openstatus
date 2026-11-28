package testutil

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	playbookv1connect "buf.build/gen/go/openstatus/api/connectrpc/go/openstatus/playbook/v1/playbookv1connect"
	playbookv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/playbook/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
)

// The fake mirrors the API's playbook rules (apps/server and
// packages/services/src/playbook in openstatusHQ/openstatus) independently of
// the provider, so tests catch the provider disagreeing with the server.

// fakeBuiltinOrder is the order the API lists built-ins in, ahead of custom
// playbooks.
var fakeBuiltinOrder = []string{"incident-response", "status-page", "maintenance-windows", "postmortem"}

var fakeBuiltinPlaybooks = map[string][2]string{
	"incident-response":   {"Incident response", "How you run an incident."},
	"status-page":         {"Status page", "How you write on the status page."},
	"maintenance-windows": {"Maintenance windows", "How a maintenance window differs from a maintenance."},
	"postmortem":          {"Postmortem", "The sections of your postmortems."},
}

var fakeVariableRef = regexp.MustCompile(`\{\{\s*([a-z0-9_]{1,64})\s*\}\}`)

// fakeStoredVariables keeps the trimmed, non-empty values of the variables
// the content uses, and drops everything else, like the API does.
func fakeStoredVariables(content string, variables map[string]string) map[string]string {
	out := map[string]string{}
	for _, m := range fakeVariableRef.FindAllStringSubmatch(content, -1) {
		if v := strings.TrimSpace(variables[m[1]]); v != "" {
			out[m[1]] = v
		}
	}
	return out
}

// Playbook returns a copy of the stored playbook, or nil when the slug has
// none. It is a copy so test checks never race with the server's handlers.
func (f *Fake) Playbook(slug string) *playbookv1.Playbook {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.playbooks[slug]
	if !ok {
		return nil
	}
	return proto.CloneOf(p)
}

// SeedPlaybook stores a playbook as if the workspace had written it in the
// dashboard, outside Terraform.
func (f *Fake) SeedPlaybook(slug, title, description, content string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	(&playbookService{f: f}).insert(slug, title, description, content, nil)
}

type playbookService struct {
	playbookv1connect.UnimplementedPlaybookServiceHandler
	f *Fake
}

func (s *playbookService) insert(slug, title, description, content string, variables map[string]string) *playbookv1.Playbook {
	p := &playbookv1.Playbook{}
	p.SetSlug(slug)
	p.SetTitle(strings.TrimSpace(title))
	p.SetDescription(strings.TrimSpace(description))
	p.SetContent(strings.TrimSpace(content))
	p.SetVariables(fakeStoredVariables(content, variables))
	p.SetCreatedAt(fixedTimestamp)
	p.SetUpdatedAt(fixedTimestamp)
	s.f.playbooks[slug] = p
	s.f.playbookOrder = append(s.f.playbookOrder, slug)
	return p
}

func (s *playbookService) ListPlaybooks(_ context.Context, _ *connect.Request[playbookv1.ListPlaybooksRequest]) (*connect.Response[playbookv1.ListPlaybooksResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	// Built-ins first in their fixed order, then custom ones by age.
	var ordered []string
	for _, slug := range fakeBuiltinOrder {
		if _, ok := s.f.playbooks[slug]; ok {
			ordered = append(ordered, slug)
		}
	}
	for _, slug := range s.f.playbookOrder {
		if _, builtin := fakeBuiltinPlaybooks[slug]; !builtin {
			ordered = append(ordered, slug)
		}
	}

	summaries := make([]*playbookv1.PlaybookSummary, 0, len(ordered))
	for _, slug := range ordered {
		p := s.f.playbooks[slug]
		summary := &playbookv1.PlaybookSummary{}
		summary.SetSlug(p.GetSlug())
		summary.SetTitle(p.GetTitle())
		summary.SetDescription(p.GetDescription())
		summary.SetCreatedAt(p.GetCreatedAt())
		summary.SetUpdatedAt(p.GetUpdatedAt())
		summaries = append(summaries, summary)
	}

	resp := &playbookv1.ListPlaybooksResponse{}
	resp.SetPlaybooks(summaries)
	return connect.NewResponse(resp), nil
}

func (s *playbookService) GetPlaybook(_ context.Context, req *connect.Request[playbookv1.GetPlaybookRequest]) (*connect.Response[playbookv1.GetPlaybookResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	p, ok := s.f.playbooks[req.Msg.GetSlug()]
	if !ok {
		return nil, notFound("playbook", req.Msg.GetSlug())
	}
	resp := &playbookv1.GetPlaybookResponse{}
	resp.SetPlaybook(proto.CloneOf(p))
	return connect.NewResponse(resp), nil
}

func (s *playbookService) CreatePlaybook(_ context.Context, req *connect.Request[playbookv1.CreatePlaybookRequest]) (*connect.Response[playbookv1.CreatePlaybookResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	m := req.Msg
	if _, ok := fakeBuiltinPlaybooks[m.GetSlug()]; ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%s is a built-in playbook; update it instead", m.GetSlug()))
	}
	if strings.TrimSpace(m.GetTitle()) == "" || strings.TrimSpace(m.GetDescription()) == "" || strings.TrimSpace(m.GetContent()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("title, description and content are required"))
	}
	if _, ok := s.f.playbooks[m.GetSlug()]; ok {
		return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("a %s playbook already exists", m.GetSlug()))
	}

	p := s.insert(m.GetSlug(), m.GetTitle(), m.GetDescription(), m.GetContent(), m.GetVariables())

	resp := &playbookv1.CreatePlaybookResponse{}
	resp.SetPlaybook(proto.CloneOf(p))
	return connect.NewResponse(resp), nil
}

func (s *playbookService) UpdatePlaybook(_ context.Context, req *connect.Request[playbookv1.UpdatePlaybookRequest]) (*connect.Response[playbookv1.UpdatePlaybookResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	m := req.Msg
	if strings.TrimSpace(m.GetContent()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("a playbook cannot be empty"))
	}

	resp := &playbookv1.UpdatePlaybookResponse{}

	p, ok := s.f.playbooks[m.GetSlug()]
	if !ok {
		defaults, builtin := fakeBuiltinPlaybooks[m.GetSlug()]
		if !builtin {
			return nil, notFound("playbook", m.GetSlug())
		}
		title, description := defaults[0], defaults[1]
		if m.HasTitle() {
			title = m.GetTitle()
		}
		if m.HasDescription() {
			description = m.GetDescription()
		}
		resp.SetPlaybook(proto.CloneOf(s.insert(m.GetSlug(), title, description, m.GetContent(), m.GetVariables())))
		return connect.NewResponse(resp), nil
	}

	if m.HasTitle() {
		p.SetTitle(strings.TrimSpace(m.GetTitle()))
	}
	if m.HasDescription() {
		p.SetDescription(strings.TrimSpace(m.GetDescription()))
	}
	p.SetContent(strings.TrimSpace(m.GetContent()))
	p.SetVariables(fakeStoredVariables(m.GetContent(), m.GetVariables()))
	p.SetUpdatedAt("2026-01-02T00:00:00Z")

	resp.SetPlaybook(proto.CloneOf(p))
	return connect.NewResponse(resp), nil
}

func (s *playbookService) DeletePlaybook(_ context.Context, req *connect.Request[playbookv1.DeletePlaybookRequest]) (*connect.Response[playbookv1.DeletePlaybookResponse], error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()

	slug := req.Msg.GetSlug()
	if _, ok := s.f.playbooks[slug]; !ok {
		return nil, notFound("playbook", slug)
	}
	delete(s.f.playbooks, slug)
	s.f.playbookOrder = slices.DeleteFunc(s.f.playbookOrder, func(v string) bool { return v == slug })

	resp := &playbookv1.DeletePlaybookResponse{}
	resp.SetSuccess(true)
	return connect.NewResponse(resp), nil
}
