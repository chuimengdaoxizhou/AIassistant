import grpc
from concurrent import futures
import time
import etcd3
import json
import pandas as pd
import numpy as np
from io import StringIO, BytesIO
import logging
import matplotlib.pyplot as plt

import agent_task_pb2
import agent_task_pb2_grpc

# --- 配置 ---
SERVICE_NAME = "data_analysis_agent"
SERVICE_ADDRESS = "localhost:9092"  # Agent 监听的地址和端口
ETCD_HOST = "localhost"
ETCD_PORT = 2379
LEASE_TTL = 10  # etcd 租约的 TTL (秒)

# --- 日志配置 ---
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(SERVICE_NAME)

class AgentServiceServicer(agent_task_pb2_grpc.AgentServiceServicer):
    """实现了 AgentService gRPC 服务的类"""

    def GetMetadata(self, request, context):
        logger.info("GetMetadata called")
        return agent_task_pb2.AgentMetadata(
            name=SERVICE_NAME,
            capability="提供强大的数据分析和可视化能力。可以读取CSV或JSON数据，执行多种分析任务（如描述性统计、相关性分析、缺失值检查、值计数）和绘图任务（折线图、柱状图、直方图、散点图）。",
            input_description='''一个JSON对象，包含三个键：
1. `data`: CSV或JSON格式的字符串数据。
2. `data_format`: "csv" 或 "json"。
3. `tasks`: 一个任务列表，每个任务是一个JSON对象，如 `{"type": "describe"}` 或 `{"type": "plot", "plot_type": "line", "x": "col1", "y": "col2"}`。
''',
            output_description="返回一个多部分任务结果，其中文本部分包含JSON格式的分析结果，图片部分包含PNG格式的可视化图表。"
        )

    def _load_dataframe(self, data_str: str, data_format: str) -> pd.DataFrame:
        if data_format == 'csv':
            return pd.read_csv(StringIO(data_str))
        elif data_format == 'json':
            return pd.read_json(StringIO(data_str), orient='records')
        else:
            raise ValueError(f"Unsupported data format: {data_format}")

    def _handle_describe(self, df: pd.DataFrame, params: dict) -> agent_task_pb2.Part:
        logger.info("Executing 'describe' task")
        result = df.describe().to_json(orient='split')
        return agent_task_pb2.Part(text=json.dumps({"task_type": "describe", "result": json.loads(result)}))

    def _handle_correlation(self, df: pd.DataFrame, params: dict) -> agent_task_pb2.Part:
        logger.info("Executing 'correlation' task")
        # Ensure only numeric columns are used for correlation
        numeric_df = df.select_dtypes(include=np.number)
        result = numeric_df.corr().to_json(orient='split')
        return agent_task_pb2.Part(text=json.dumps({"task_type": "correlation", "result": json.loads(result)}))

    def _handle_missing_values(self, df: pd.DataFrame, params: dict) -> agent_task_pb2.Part:
        logger.info("Executing 'missing_values' task")
        result = df.isnull().sum().to_json(orient='split')
        return agent_task_pb2.Part(text=json.dumps({"task_type": "missing_values", "result": json.loads(result)}))

    def _handle_value_counts(self, df: pd.DataFrame, params: dict) -> agent_task_pb2.Part:
        column = params.get('column')
        if not column:
            raise ValueError("'value_counts' task requires a 'column' parameter.")
        logger.info(f"Executing 'value_counts' task for column: {column}")
        result = df[column].value_counts().to_json(orient='split')
        return agent_task_pb2.Part(text=json.dumps({"task_type": "value_counts", "column": column, "result": json.loads(result)}))

    def _handle_plot(self, df: pd.DataFrame, params: dict) -> agent_task_pb2.Part:
        plot_type = params.get('plot_type')
        logger.info(f"Executing 'plot' task with type: {plot_type}")

        plt.style.use('seaborn-v0_8-whitegrid')
        fig, ax = plt.subplots(figsize=(10, 6))

        if plot_type == 'line':
            ax.plot(df[params['x']], df[params['y']])
            ax.set_xlabel(params['x'])
            ax.set_ylabel(params['y'])
            ax.set_title(f"Line Plot of {params['y']} vs {params['x']}")
        elif plot_type == 'bar':
            ax.bar(df[params['x']], df[params['y']])
            ax.set_xlabel(params['x'])
            ax.set_ylabel(params['y'])
            ax.set_title(f"Bar Chart of {params['y']} vs {params['x']}")
        elif plot_type == 'histogram':
            ax.hist(df[params['column']], bins=params.get('bins', 10))
            ax.set_xlabel(params['column'])
            ax.set_ylabel("Frequency")
            ax.set_title(f"Histogram of {params['column']}")
        elif plot_type == 'scatter':
            ax.scatter(df[params['x']], df[params['y']])
            ax.set_xlabel(params['x'])
            ax.set_ylabel(params['y'])
            ax.set_title(f"Scatter Plot of {params['y']} vs {params['x']}")
        else:
            raise ValueError(f"Unsupported plot type: {plot_type}")
        
        ax.tick_params(axis='x', rotation=45)
        fig.tight_layout()

        buf = BytesIO()
        fig.savefig(buf, format='png')
        plt.close(fig)
        buf.seek(0)

        return agent_task_pb2.Part(
            inline_data=agent_task_pb2.Blob(
                mime_type='image/png',
                data=buf.read()
            )
        )

    def ExecuteTask(self, request, context):
        logger.info(f"ExecuteTask called with task: {request.task_name}")
        try:
            input_text = request.content[0].parts[0].text
            logger.info(f"Received input text length: {len(input_text)}")
            
            try:
                input_payload = json.loads(input_text)
                if 'input' in input_payload: # Compatibility with Go agent wrapping
                    task_spec = json.loads(input_payload['input'])
                else:
                    task_spec = input_payload
            except json.JSONDecodeError:
                raise ValueError("Input is not a valid JSON string.")

            data_str = task_spec.get('data')
            data_format = task_spec.get('data_format', 'csv')
            tasks = task_spec.get('tasks', [])

            if not data_str or not tasks:
                raise ValueError("'data' and 'tasks' fields are required in the input JSON.")

            df = self._load_dataframe(data_str, data_format)
            
            result_parts = []
            task_handlers = {
                "describe": self._handle_describe,
                "correlation": self._handle_correlation,
                "missing_values": self._handle_missing_values,
                "value_counts": self._handle_value_counts,
                "plot": self._handle_plot
            }

            for task in tasks:
                task_type = task.get('type')
                handler = task_handlers.get(task_type)
                if handler:
                    part = handler(df, task)
                    result_parts.append(part)
                else:
                    logger.warning(f"Unknown task type: {task_type}")

            result_task = agent_task_pb2.AgentTask(
                task_id=f"result-{request.task_id}",
                correlation_id=request.correlation_id,
                parent_task_id=request.task_id,
                source_agent_id=SERVICE_NAME,
                target_agent_id=request.source_agent_id,
                task_name="Data Analysis Result",
                content=[agent_task_pb2.Content(parts=result_parts)]
            )
            return result_task

        except Exception as e:
            logger.error(f"Error executing task: {e}", exc_info=True)
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"An error occurred: {e}")
            return agent_task_pb2.AgentTask()


def register_service(etcd_client, service_name, service_address, ttl):
    lease = etcd_client.lease(ttl)
    key = f"/{service_name}/{service_address}"
    
    etcd_client.put(key, service_address, lease.id)
    logger.info(f"Service '{service_name}' registered at '{key}' with TTL {ttl}s.")

    def refresh_lease():
        while True:
            try:
                lease.refresh()
                time.sleep(ttl / 2)
            except Exception as e:
                logger.error(f"Failed to refresh lease: {e}. Re-registering...")
                try:
                    lease = etcd_client.lease(ttl)
                    etcd_client.put(key, service_address, lease.id)
                    logger.info("Service re-registered successfully.")
                except Exception as reg_e:
                    logger.error(f"Failed to re-register service: {reg_e}")
                    time.sleep(5)

    refresh_thread = futures.ThreadPoolExecutor(max_workers=1)
    refresh_thread.submit(refresh_lease)
    logger.info("Lease refresh thread started.")

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    agent_task_pb2_grpc.add_AgentServiceServicer_to_server(AgentServiceServicer(), server)
    server.add_insecure_port(SERVICE_ADDRESS)
    
    try:
        etcd_client = etcd3.client(host=ETCD_HOST, port=ETCD_PORT)
        etcd_client.status()
        logger.info("Successfully connected to etcd.")
    except Exception as e:
        logger.error(f"Failed to connect to etcd: {e}")
        return

    register_service(etcd_client, SERVICE_NAME, SERVICE_ADDRESS, LEASE_TTL)

    server.start()
    logger.info(f"Server started, listening on {SERVICE_ADDRESS}")
    
    try:
        while True:
            time.sleep(86400)
    except KeyboardInterrupt:
        logger.info("Server stopping...")
        server.stop(0)
        etcd_client.delete(f"/{SERVICE_NAME}/{SERVICE_ADDRESS}")
        logger.info("Service unregistered from etcd.")

if __name__ == '__main__':
    serve()