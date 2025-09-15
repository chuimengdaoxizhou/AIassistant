
package models

import (
	v1 "Jarvis_2.0/api/proto/v1"
	"github.com/google/generative-ai-go/genai"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

// ConvertAgentMetadataToFunctionDeclarations 根据您的新思路：
// 将从子Agent获取的简单元数据（proto类型）合成为LLM所需的、结构化的FunctionDeclaration。
// 每个子Agent都被包装成一个接收单一字符串参数（task_description）的工具。
func ConvertAgentMetadataToFunctionDeclarations(metadataList []*v1.AgentMetadata) []*genai.FunctionDeclaration {
	if metadataList == nil {
		return nil
	}
	declarations := make([]*genai.FunctionDeclaration, 0, len(metadataList))
	for _, meta := range metadataList {
		// 为每个 agent 创建一个标准化的输入参数 Schema
		params := &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"task_description": {
					Type:        genai.TypeString,
					Description: meta.InputDescription, // 使用proto中的 input_description 作为参数描述
				},
			},
			Required: []string{"task_description"},
		}

		declarations = append(declarations, &genai.FunctionDeclaration{
			Name:        meta.Name,       // 使用 proto 中的 name
			Description: meta.Capability, // 使用 proto 中的 capability 作为工具描述
			Parameters:  params,
		})
	}
	return declarations
}

// --- 其他转换函数保持不变 ---

// ConvertProtoToModelsContent 将 protobuf 的 Content 转换为 models 的 Content。
func ConvertProtoToModelsContent(protoContent []*v1.Content) []Content {
	if protoContent == nil {
		return nil
	}
	modelContent := make([]Content, len(protoContent))
	for i, pc := range protoContent {
		modelContent[i] = Content{
			Role:  SpeakerRole(pc.Role),
			Parts: ConvertProtoToModelsParts(pc.Parts),
		}
	}
	return modelContent
}

// ConvertProtoToModelsParts 将 protobuf 的 Part 转换为 models 的 Part。
func ConvertProtoToModelsParts(protoParts []*v1.Part) []*Part {
	if protoParts == nil {
		return nil
	}
	modelParts := make([]*Part, len(protoParts))
	for i, pp := range protoParts {
		modelParts[i] = &Part{
			Text:                pp.Text,
			Thought:             pp.Thought,
			ThoughtSignature:    pp.ThoughtSignature,
			InlineData:          ConvertProtoToModelsBlob(pp.InlineData),
			FileData:            ConvertProtoToModelsFileData(pp.FileData),
			FunctionCall:        ConvertProtoToModelsFunctionCall(pp.FunctionCall),
			FunctionResponse:    ConvertProtoToModelsFunctionResponse(pp.FunctionResponse),
			CodeExecutionResult: ConvertProtoToModelsCodeExecutionResult(pp.CodeExecutionResult),
			ExecutableCode:      ConvertProtoToModelsExecutableCode(pp.ExecutableCode),
			VideoMetadata:       ConvertProtoToModelsVideoMetadata(pp.VideoMetadata),
		}
	}
	return modelParts
}

// ConvertProtoToModelsFunctionCall 将 protobuf 的 FunctionCall 转换为 models 的 FunctionCall。
func ConvertProtoToModelsFunctionCall(protoFC *v1.FunctionCall) *FunctionCall {
	if protoFC == nil {
		return nil
	}
	return &FunctionCall{
		ID:   protoFC.Id,
		Name: protoFC.Name,
		Args: protoFC.Args.AsMap(),
	}
}

// ConvertModelsToProtoContent 将 models 的 Content 转换为 protobuf 的 Content。
func ConvertModelsToProtoContent(modelContent []Content) ([]*v1.Content, error) {
	if modelContent == nil {
		return nil, nil
	}
	protoContent := make([]*v1.Content, len(modelContent))
	for i, mc := range modelContent {
		protoParts, err := ConvertModelsToProtoParts(mc.Parts)
		if err != nil {
			return nil, err
		}
		protoContent[i] = &v1.Content{
			Role:  string(mc.Role),
			Parts: protoParts,
		}
	}
	return protoContent, nil
}

// ConvertModelsToProtoParts 将 models 的 Part 转换为 protobuf 的 Part。
func ConvertModelsToProtoParts(modelParts []*Part) ([]*v1.Part, error) {
	if modelParts == nil {
		return nil, nil
	}
	protoParts := make([]*v1.Part, len(modelParts))
	for i, mp := range modelParts {
		var err error
		protoPart := &v1.Part{
			Thought:          mp.Thought,
			ThoughtSignature: mp.ThoughtSignature,
		}

		switch true {
		case mp.FunctionCall != nil:
			protoPart.FunctionCall, err = ConvertModelsToProtoFunctionCall(mp.FunctionCall)
		case mp.FunctionResponse != nil:
			protoPart.FunctionResponse, err = ConvertModelsToProtoFunctionResponse(mp.FunctionResponse)
		case mp.InlineData != nil:
			protoPart.InlineData = ConvertModelsToProtoBlob(mp.InlineData)
		case mp.FileData != nil:
			protoPart.FileData = ConvertModelsToProtoFileData(mp.FileData)
		case mp.CodeExecutionResult != nil:
			protoPart.CodeExecutionResult = ConvertModelsToProtoCodeExecutionResult(mp.CodeExecutionResult)
		case mp.ExecutableCode != nil:
			protoPart.ExecutableCode = ConvertModelsToProtoExecutableCode(mp.ExecutableCode)
		case mp.VideoMetadata != nil:
			protoPart.VideoMetadata = ConvertModelsToProtoVideoMetadata(mp.VideoMetadata)
		default:
			protoPart.Text = mp.Text
		}

		if err != nil {
			return nil, err
		}
		protoParts[i] = protoPart
	}
	return protoParts, nil
}

// ConvertModelsToProtoFunctionCall 将 models 的 FunctionCall 转换为 protobuf 的 FunctionCall。
func ConvertModelsToProtoFunctionCall(modelFC *FunctionCall) (*v1.FunctionCall, error) {
	if modelFC == nil {
		return nil, nil
	}
	args, err := structpb.NewStruct(modelFC.Args)
	if err != nil {
		return nil, err
	}
	return &v1.FunctionCall{
		Id:   modelFC.ID,
		Name: modelFC.Name,
		Args: args,
	},
	 nil
}

// ConvertModelsToProtoTask 根据父任务为子任务创建一个新的 AgentTask。
func ConvertModelsToProtoTask(parentTask *v1.AgentTask, newContent Content, targetAgentID, taskName string) (*v1.AgentTask, error) {
	protoContent, err := ConvertModelsToProtoContent([]Content{newContent})
	if err != nil {
		return nil, err
	}

	newTaskId := uuid.NewString()
	correlationId := parentTask.GetCorrelationId()
	if correlationId == "" {
		correlationId = uuid.NewString()
	}

	subTask := &v1.AgentTask{
		TaskId:        newTaskId,
		CorrelationId: correlationId,
		ParentTaskId:  parentTask.TaskId,
		SourceAgentId: parentTask.TargetAgentId,
		TargetAgentId: targetAgentID,
		TaskName:      taskName,
		Content:       protoContent,
		CreatedAt:     timestamppb.New(time.Now()),
		TimeoutSeconds: parentTask.TimeoutSeconds,
		RetryPolicy:    parentTask.RetryPolicy,
	}

	return subTask, nil
}

// --- 其他辅助转换函数 ---

func ConvertProtoToModelsBlob(protoBlob *v1.Blob) *Blob {
	if protoBlob == nil {
		return nil
	}
	return &Blob{
		DisplayName: protoBlob.DisplayName,
		Data:        protoBlob.Data,
		MIMEType:    protoBlob.MimeType,
	}
}

func ConvertModelsToProtoBlob(modelBlob *Blob) *v1.Blob {
	if modelBlob == nil {
		return nil
	}
	return &v1.Blob{
		DisplayName: modelBlob.DisplayName,
		Data:        modelBlob.Data,
		MimeType:    modelBlob.MIMEType,
	}
}

func ConvertProtoToModelsFileData(protoFileData *v1.FileData) *FileData {
	if protoFileData == nil {
		return nil
	}
	return &FileData{
		DisplayName: protoFileData.DisplayName,
		FileURI:     protoFileData.FileUri,
		MIMEType:    protoFileData.MimeType,
	}
}

func ConvertModelsToProtoFileData(modelFileData *FileData) *v1.FileData {
	if modelFileData == nil {
		return nil
	}
	return &v1.FileData{
		DisplayName: modelFileData.DisplayName,
		FileUri:     modelFileData.FileURI,
		MimeType:    modelFileData.MIMEType,
	}
}

func ConvertProtoToModelsFunctionResponse(protoFR *v1.FunctionResponse) *FunctionResponse {
	if protoFR == nil {
		return nil
	}
	return &FunctionResponse{
		WillContinue: protoFR.WillContinue,
		Scheduling:   FunctionResponseScheduling(protoFR.Scheduling.String()),
		ID:           protoFR.Id,
		Name:         protoFR.Name,
		Response:     protoFR.Response.AsMap(),
	}
}

func ConvertModelsToProtoFunctionResponse(modelFR *FunctionResponse) (*v1.FunctionResponse, error) {
	if modelFR == nil {
		return nil, nil
	}
	response, err := structpb.NewStruct(modelFR.Response)
	if err != nil {
		return nil, err
	}
	scheduling, ok := v1.FunctionResponse_FunctionResponseScheduling_value[string(modelFR.Scheduling)]
	if !ok {
		scheduling = int32(v1.FunctionResponse_SCHEDULING_UNSPECIFIED)
	}
	return &v1.FunctionResponse{
		WillContinue: modelFR.WillContinue,
		Scheduling:   v1.FunctionResponse_FunctionResponseScheduling(scheduling),
		Id:           modelFR.ID,
		Name:         modelFR.Name,
		Response:     response,
	}, nil
}

func ConvertProtoToModelsCodeExecutionResult(protoCER *v1.CodeExecutionResult) *CodeExecutionResult {
	if protoCER == nil {
		return nil
	}
	return &CodeExecutionResult{
		Outcome: Outcome(protoCER.Outcome.String()),
		Output:  protoCER.Output,
	}
}

func ConvertModelsToProtoCodeExecutionResult(modelCER *CodeExecutionResult) *v1.CodeExecutionResult {
	if modelCER == nil {
		return nil
	}
	outcome, ok := v1.CodeExecutionResult_Outcome_value[string(modelCER.Outcome)]
	if !ok {
		outcome = int32(v1.CodeExecutionResult_OUTCOME_UNSPECIFIED)
	}
	return &v1.CodeExecutionResult{
		Outcome: v1.CodeExecutionResult_Outcome(outcome),
		Output:  modelCER.Output,
	}
}

func ConvertProtoToModelsExecutableCode(protoEC *v1.ExecutableCode) *ExecutableCode {
	if protoEC == nil {
		return nil
	}
	return &ExecutableCode{
		Code:     protoEC.Code,
		Language: Language(protoEC.Language.String()),
	}
}

func ConvertModelsToProtoExecutableCode(modelEC *ExecutableCode) *v1.ExecutableCode {
	if modelEC == nil {
		return nil
	}
	language, ok := v1.ExecutableCode_Language_value[string(modelEC.Language)]
	if !ok {
		language = int32(v1.ExecutableCode_LANGUAGE_UNSPECIFIED)
	}
	return &v1.ExecutableCode{
		Code:     modelEC.Code,
		Language: v1.ExecutableCode_Language(language),
	}
}

func ConvertProtoToModelsVideoMetadata(protoVM *v1.VideoMetadata) *VideoMetadata {
	if protoVM == nil {
		return nil
	}
	return &VideoMetadata{
		FPS:         protoVM.Fps,
		EndOffset:   protoVM.EndOffset.AsDuration(),
		StartOffset: protoVM.StartOffset.AsDuration(),
	}
}

func ConvertModelsToProtoVideoMetadata(modelVM *VideoMetadata) *v1.VideoMetadata {
	if modelVM == nil {
		return nil
	}
	return &v1.VideoMetadata{
		Fps:         modelVM.FPS,
		EndOffset:   durationpb.New(modelVM.EndOffset),
		StartOffset: durationpb.New(modelVM.StartOffset),
	}
}
