package models

import (
	v1 "Jarvis_2.0/api/proto/v1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

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
			// 如果没有其他数据类型，则默认为文本。
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
	}, nil
}

// ConvertModelsToProtoTask 根据父任务为子任务创建一个新的 AgentTask。
// 这个函数是 Agent 间任务派发的核心，它通过 parentTask 传递上下文，
// 并为子任务生成新的唯一标识，同时保留了任务链的跟踪信息。
func ConvertModelsToProtoTask(parentTask *v1.AgentTask, newContent Content, targetAgentID, taskName string) (*v1.AgentTask, error) {
	protoContent, err := ConvertModelsToProtoContent([]Content{newContent})
	if err != nil {
		return nil, err
	}

	// 为子任务生成新的唯一 TaskId
	newTaskId := uuid.NewString()

	// 确保 CorrelationId 存在，如果父任务没有，则为整个链创建一个新的
	correlationId := parentTask.GetCorrelationId()
	if correlationId == "" {
		correlationId = uuid.NewString()
	}

	// 构建新的 AgentTask
	subTask := &v1.AgentTask{
		// --- 核心元数据 ---
		TaskId:        newTaskId,
		CorrelationId: correlationId,      // 传递 CorrelationId 以便进行端到端跟踪
		ParentTaskId:  parentTask.TaskId, // 设置 ParentTaskId 以建立父子关系

		// --- 路由和命名 ---
		SourceAgentId: parentTask.TargetAgentId, // 源 Agent 是当前 Agent
		TargetAgentId: targetAgentID,           // 目标是子 Agent
		TaskName:      taskName,

		// --- 载荷 (Payload) ---
		Content: protoContent,

		// --- 控制参数 ---
		CreatedAt: timestamppb.New(time.Now()),
		// Timeout 和 RetryPolicy 可以从父任务继承或根据需要重新定义
		TimeoutSeconds: parentTask.TimeoutSeconds,
		RetryPolicy:    parentTask.RetryPolicy,
	}

	return subTask, nil
}

// --- 新增的辅助转换函数 ---

// ConvertProtoToModelsBlob 将 protobuf 的 Blob 转换为 models 的 Blob。
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

// ConvertModelsToProtoBlob 将 models 的 Blob 转换为 protobuf 的 Blob。
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

// ConvertProtoToModelsFileData 将 protobuf 的 FileData 转换为 models 的 FileData。
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

// ConvertModelsToProtoFileData 将 models 的 FileData 转换为 protobuf 的 FileData。
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

// ConvertProtoToModelsFunctionResponse 将 protobuf 的 FunctionResponse 转换为 models 的 FunctionResponse。
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

// ConvertModelsToProtoFunctionResponse 将 models 的 FunctionResponse 转换为 protobuf 的 FunctionResponse。
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

// ConvertProtoToModelsCodeExecutionResult 将 protobuf 的 CodeExecutionResult 转换为 models 的 CodeExecutionResult。
func ConvertProtoToModelsCodeExecutionResult(protoCER *v1.CodeExecutionResult) *CodeExecutionResult {
	if protoCER == nil {
		return nil
	}
	return &CodeExecutionResult{
		Outcome: Outcome(protoCER.Outcome.String()),
		Output:  protoCER.Output,
	}
}

// ConvertModelsToProtoCodeExecutionResult 将 models 的 CodeExecutionResult 转换为 protobuf 的 CodeExecutionResult。
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

// ConvertProtoToModelsExecutableCode 将 protobuf 的 ExecutableCode 转换为 models 的 ExecutableCode。
func ConvertProtoToModelsExecutableCode(protoEC *v1.ExecutableCode) *ExecutableCode {
	if protoEC == nil {
		return nil
	}
	return &ExecutableCode{
		Code:     protoEC.Code,
		Language: Language(protoEC.Language.String()),
	}
}

// ConvertModelsToProtoExecutableCode 将 models 的 ExecutableCode 转换为 protobuf 的 ExecutableCode。
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

// ConvertProtoToModelsVideoMetadata 将 protobuf 的 VideoMetadata 转换为 models 的 VideoMetadata。
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

// ConvertModelsToProtoVideoMetadata 将 models 的 VideoMetadata 转换为 protobuf 的 VideoMetadata。
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
