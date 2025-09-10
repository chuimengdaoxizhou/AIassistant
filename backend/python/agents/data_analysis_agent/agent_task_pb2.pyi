import datetime

from google.protobuf import duration_pb2 as _duration_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import empty_pb2 as _empty_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class VideoMetadata(_message.Message):
    __slots__ = ("fps", "end_offset", "start_offset")
    FPS_FIELD_NUMBER: _ClassVar[int]
    END_OFFSET_FIELD_NUMBER: _ClassVar[int]
    START_OFFSET_FIELD_NUMBER: _ClassVar[int]
    fps: float
    end_offset: _duration_pb2.Duration
    start_offset: _duration_pb2.Duration
    def __init__(self, fps: _Optional[float] = ..., end_offset: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., start_offset: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ...) -> None: ...

class Blob(_message.Message):
    __slots__ = ("display_name", "data", "mime_type")
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    MIME_TYPE_FIELD_NUMBER: _ClassVar[int]
    display_name: str
    data: bytes
    mime_type: str
    def __init__(self, display_name: _Optional[str] = ..., data: _Optional[bytes] = ..., mime_type: _Optional[str] = ...) -> None: ...

class FileData(_message.Message):
    __slots__ = ("display_name", "file_uri", "mime_type")
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    FILE_URI_FIELD_NUMBER: _ClassVar[int]
    MIME_TYPE_FIELD_NUMBER: _ClassVar[int]
    display_name: str
    file_uri: str
    mime_type: str
    def __init__(self, display_name: _Optional[str] = ..., file_uri: _Optional[str] = ..., mime_type: _Optional[str] = ...) -> None: ...

class CodeExecutionResult(_message.Message):
    __slots__ = ("outcome", "output")
    class Outcome(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        OUTCOME_UNSPECIFIED: _ClassVar[CodeExecutionResult.Outcome]
        OK: _ClassVar[CodeExecutionResult.Outcome]
        FAILED: _ClassVar[CodeExecutionResult.Outcome]
    OUTCOME_UNSPECIFIED: CodeExecutionResult.Outcome
    OK: CodeExecutionResult.Outcome
    FAILED: CodeExecutionResult.Outcome
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    outcome: CodeExecutionResult.Outcome
    output: str
    def __init__(self, outcome: _Optional[_Union[CodeExecutionResult.Outcome, str]] = ..., output: _Optional[str] = ...) -> None: ...

class ExecutableCode(_message.Message):
    __slots__ = ("code", "language")
    class Language(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        LANGUAGE_UNSPECIFIED: _ClassVar[ExecutableCode.Language]
        PYTHON: _ClassVar[ExecutableCode.Language]
        GO: _ClassVar[ExecutableCode.Language]
    LANGUAGE_UNSPECIFIED: ExecutableCode.Language
    PYTHON: ExecutableCode.Language
    GO: ExecutableCode.Language
    CODE_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    code: str
    language: ExecutableCode.Language
    def __init__(self, code: _Optional[str] = ..., language: _Optional[_Union[ExecutableCode.Language, str]] = ...) -> None: ...

class FunctionCall(_message.Message):
    __slots__ = ("id", "args", "name")
    ID_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    id: str
    args: _struct_pb2.Struct
    name: str
    def __init__(self, id: _Optional[str] = ..., args: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., name: _Optional[str] = ...) -> None: ...

class FunctionResponse(_message.Message):
    __slots__ = ("will_continue", "scheduling", "id", "name", "response")
    class FunctionResponseScheduling(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        SCHEDULING_UNSPECIFIED: _ClassVar[FunctionResponse.FunctionResponseScheduling]
        WHEN_IDLE: _ClassVar[FunctionResponse.FunctionResponseScheduling]
        IMMEDIATE: _ClassVar[FunctionResponse.FunctionResponseScheduling]
    SCHEDULING_UNSPECIFIED: FunctionResponse.FunctionResponseScheduling
    WHEN_IDLE: FunctionResponse.FunctionResponseScheduling
    IMMEDIATE: FunctionResponse.FunctionResponseScheduling
    WILL_CONTINUE_FIELD_NUMBER: _ClassVar[int]
    SCHEDULING_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_FIELD_NUMBER: _ClassVar[int]
    will_continue: bool
    scheduling: FunctionResponse.FunctionResponseScheduling
    id: str
    name: str
    response: _struct_pb2.Struct
    def __init__(self, will_continue: bool = ..., scheduling: _Optional[_Union[FunctionResponse.FunctionResponseScheduling, str]] = ..., id: _Optional[str] = ..., name: _Optional[str] = ..., response: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class Part(_message.Message):
    __slots__ = ("video_metadata", "thought", "inline_data", "file_data", "thought_signature", "code_execution_result", "executable_code", "function_call", "function_response", "text")
    VIDEO_METADATA_FIELD_NUMBER: _ClassVar[int]
    THOUGHT_FIELD_NUMBER: _ClassVar[int]
    INLINE_DATA_FIELD_NUMBER: _ClassVar[int]
    FILE_DATA_FIELD_NUMBER: _ClassVar[int]
    THOUGHT_SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    CODE_EXECUTION_RESULT_FIELD_NUMBER: _ClassVar[int]
    EXECUTABLE_CODE_FIELD_NUMBER: _ClassVar[int]
    FUNCTION_CALL_FIELD_NUMBER: _ClassVar[int]
    FUNCTION_RESPONSE_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    video_metadata: VideoMetadata
    thought: bool
    inline_data: Blob
    file_data: FileData
    thought_signature: bytes
    code_execution_result: CodeExecutionResult
    executable_code: ExecutableCode
    function_call: FunctionCall
    function_response: FunctionResponse
    text: str
    def __init__(self, video_metadata: _Optional[_Union[VideoMetadata, _Mapping]] = ..., thought: bool = ..., inline_data: _Optional[_Union[Blob, _Mapping]] = ..., file_data: _Optional[_Union[FileData, _Mapping]] = ..., thought_signature: _Optional[bytes] = ..., code_execution_result: _Optional[_Union[CodeExecutionResult, _Mapping]] = ..., executable_code: _Optional[_Union[ExecutableCode, _Mapping]] = ..., function_call: _Optional[_Union[FunctionCall, _Mapping]] = ..., function_response: _Optional[_Union[FunctionResponse, _Mapping]] = ..., text: _Optional[str] = ...) -> None: ...

class Content(_message.Message):
    __slots__ = ("parts", "role")
    PARTS_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    parts: _containers.RepeatedCompositeFieldContainer[Part]
    role: str
    def __init__(self, parts: _Optional[_Iterable[_Union[Part, _Mapping]]] = ..., role: _Optional[str] = ...) -> None: ...

class GenerateContentRequest(_message.Message):
    __slots__ = ("content", "role")
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    content: _containers.RepeatedCompositeFieldContainer[Content]
    role: str
    def __init__(self, content: _Optional[_Iterable[_Union[Content, _Mapping]]] = ..., role: _Optional[str] = ...) -> None: ...

class GenerateContentResponse(_message.Message):
    __slots__ = ("content", "create_time", "response_id", "model_version")
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    CREATE_TIME_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_ID_FIELD_NUMBER: _ClassVar[int]
    MODEL_VERSION_FIELD_NUMBER: _ClassVar[int]
    content: _containers.RepeatedCompositeFieldContainer[Content]
    create_time: _timestamp_pb2.Timestamp
    response_id: str
    model_version: str
    def __init__(self, content: _Optional[_Iterable[_Union[Content, _Mapping]]] = ..., create_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., response_id: _Optional[str] = ..., model_version: _Optional[str] = ...) -> None: ...

class RetryPolicy(_message.Message):
    __slots__ = ("max_retries", "backoff_coeff", "initial_delay")
    MAX_RETRIES_FIELD_NUMBER: _ClassVar[int]
    BACKOFF_COEFF_FIELD_NUMBER: _ClassVar[int]
    INITIAL_DELAY_FIELD_NUMBER: _ClassVar[int]
    max_retries: int
    backoff_coeff: float
    initial_delay: str
    def __init__(self, max_retries: _Optional[int] = ..., backoff_coeff: _Optional[float] = ..., initial_delay: _Optional[str] = ...) -> None: ...

class AgentTask(_message.Message):
    __slots__ = ("task_id", "correlation_id", "parent_task_id", "source_agent_id", "target_agent_id", "task_name", "content", "created_at", "timeout_seconds", "retry_policy")
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    CORRELATION_ID_FIELD_NUMBER: _ClassVar[int]
    PARENT_TASK_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_NAME_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    RETRY_POLICY_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    correlation_id: str
    parent_task_id: str
    source_agent_id: str
    target_agent_id: str
    task_name: str
    content: _containers.RepeatedCompositeFieldContainer[Content]
    created_at: _timestamp_pb2.Timestamp
    timeout_seconds: int
    retry_policy: RetryPolicy
    def __init__(self, task_id: _Optional[str] = ..., correlation_id: _Optional[str] = ..., parent_task_id: _Optional[str] = ..., source_agent_id: _Optional[str] = ..., target_agent_id: _Optional[str] = ..., task_name: _Optional[str] = ..., content: _Optional[_Iterable[_Union[Content, _Mapping]]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., timeout_seconds: _Optional[int] = ..., retry_policy: _Optional[_Union[RetryPolicy, _Mapping]] = ...) -> None: ...

class AgentMetadata(_message.Message):
    __slots__ = ("name", "capability", "input_description", "output_description")
    NAME_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    INPUT_DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    name: str
    capability: str
    input_description: str
    output_description: str
    def __init__(self, name: _Optional[str] = ..., capability: _Optional[str] = ..., input_description: _Optional[str] = ..., output_description: _Optional[str] = ...) -> None: ...
