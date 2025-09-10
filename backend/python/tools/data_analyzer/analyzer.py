import json
import pandas as pd
import numpy as np
import pathlib
from mcp.server.fastmcp import FastMCP

# --- MCP Server Initialization ---
# 创建一个MCP服务器实例，并为其命名
mcp = FastMCP("ComprehensiveDataAnalyzer")

# --- Custom JSON Encoder for Numpy Types ---
class CustomJSONEncoder(json.JSONEncoder):
    """ 自定义JSON编码器，以正确处理Numpy和Pandas的数据类型 """
    def default(self, obj):
        if isinstance(obj, np.integer):
            return int(obj)
        if isinstance(obj, np.floating):
            return float(obj)
        if isinstance(obj, np.ndarray):
            return obj.tolist()
        if pd.isna(obj):
            return None
        # 处理 Pandas 的 Timestamp
        if isinstance(obj, pd.Timestamp):
            return obj.isoformat()
        return super(CustomJSONEncoder, self).default(obj)

# --- Core Analysis Logic as an MCP Tool ---
@mcp.tool()
def analyze_file(file_path: str) -> str:
    """对给定的数据文件（CSV或JSON）进行全面、自动化的初步分析。

    该工具会执行一套标准的数据科学工作流，包括：
    1. 基本信息（行数、列数）。
    2. 每列的数据类型。
    3. 缺失值统计。
    4. 对数值列的描述性统计（均值、标准差等）。
    5. 对分类列的值计数。
    6. 数值列之间的相关性矩阵。
    7. 文件头部的数据样本预览。

    :param file_path: 要分析的数据文件的绝对路径。
    :return: 一个包含所有分析结果的JSON格式字符串。
    """
    try:
        path = pathlib.Path(file_path)
        if not path.is_file():
            raise FileNotFoundError(f"文件未找到: {file_path}")

        if path.suffix == '.csv':
            df = pd.read_csv(file_path)
        elif path.suffix == '.json':
            df = pd.read_json(file_path, orient='records')
        else:
            raise ValueError(f"不支持的文件格式: {path.suffix}。请提供 .csv 或 .json 文件。")

        results = {}
        numeric_df = df.select_dtypes(include=np.number)
        categorical_cols = df.select_dtypes(include=['object', 'category']).columns

        results['basic_info'] = {'file_name': path.name, 'num_rows': len(df), 'num_columns': len(df.columns)}
        results['data_types'] = {k: str(v) for k, v in df.dtypes.to_dict().items()}
        missing_values = df.isnull().sum()
        results['missing_values'] = missing_values[missing_values > 0].to_dict()
        if not numeric_df.empty:
            results['descriptive_statistics'] = numeric_df.describe().to_dict()
        if len(categorical_cols) > 0:
            value_counts = {}
            for col in categorical_cols:
                counts = df[col].value_counts(dropna=True)
                value_counts[col] = counts.head(20).to_dict()
            results['value_counts'] = value_counts
        if len(numeric_df.columns) > 1:
            results['correlation_matrix'] = numeric_df.corr().to_dict()
        
        # 在样本数据中正确处理日期时间格式
        sample_df = df.head()
        sample_dict = sample_df.to_dict(orient='split')
        # 删除 'index' 键，因为它通常只是行号
        if 'index' in sample_dict:
            del sample_dict['index']
        results['sample_data'] = sample_dict

        final_result = {"status": "success", "analysis": results}

    except Exception as e:
        final_result = {"status": "error", "message": str(e)}
    
    # 使用自定义编码器返回一个格式化好的JSON字符串
    return json.dumps(final_result, indent=2, cls=CustomJSONEncoder)

if __name__ == "__main__":
    # 启动 SSE server，监听在 8000 端口
    mcp.run_sse(host="0.0.0.0", port=8000)

