package tools

import (
	"context"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase2 "github.com/osmcp/osmcp/docs/contracts/phase-2"
)

type fsMutationService struct {
	writeTool  *writeFileTool
	appendTool *appendFileTool
	mkdirTool  *mkdirTool
	rmTool     *rmTool
	mvTool     *mvTool
	cpTool     *cpTool
}

// NewFSMutationTool creates a new service implementing FSMutationTool
func NewFSMutationTool(policy contracts.PolicyEngine, builder contracts.EnvelopeBuilder) contracts_phase2.FSMutationTool {
	return &fsMutationService{
		writeTool:  NewWriteFileTool(policy, builder).(*writeFileTool),
		appendTool: NewAppendFileTool(policy, builder).(*appendFileTool),
		mkdirTool:  NewMkdirTool(policy, builder).(*mkdirTool),
		rmTool:     NewRmTool(policy, builder).(*rmTool),
		mvTool:     NewMvTool(policy, builder).(*mvTool),
		cpTool:     NewCpTool(policy, builder).(*cpTool),
	}
}

func (s *fsMutationService) WriteFile(req contracts_phase2.WriteFileRequest) contracts.Envelope {
	return s.writeTool.Execute(context.Background(), req)
}

func (s *fsMutationService) AppendFile(req contracts_phase2.AppendFileRequest) contracts.Envelope {
	return s.appendTool.Execute(context.Background(), req)
}

func (s *fsMutationService) Mkdir(req contracts_phase2.MkdirRequest) contracts.Envelope {
	return s.mkdirTool.Execute(context.Background(), req)
}

func (s *fsMutationService) Rm(req contracts_phase2.RmRequest) contracts.Envelope {
	return s.rmTool.Execute(context.Background(), req)
}

func (s *fsMutationService) Mv(req contracts_phase2.MvRequest) contracts.Envelope {
	return s.mvTool.Execute(context.Background(), req)
}

func (s *fsMutationService) Cp(req contracts_phase2.CpRequest) contracts.Envelope {
	return s.cpTool.Execute(context.Background(), req)
}
