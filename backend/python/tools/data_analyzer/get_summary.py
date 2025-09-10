import argparse
import pandas as pd
import json
import sys

def get_summary(file_path: str):
    """
    Reads a data file (like CSV) and returns a JSON summary.

    :param file_path: Path to the data file.
    :return: A JSON string with summary information.
    """
    try:
        df = pd.read_csv(file_path)
        summary = {
            "file_path": file_path,
            "shape": df.shape,
            "columns": list(df.columns),
            "dtypes": {col: str(dtype) for col, dtype in df.dtypes.items()},
            "memory_usage": df.memory_usage(deep=True).sum(),
        }
        print(json.dumps(summary, indent=4))
    except Exception as e:
        error_summary = {
            "error": True,
            "message": str(e),
            "file_path": file_path,
        }
        print(json.dumps(error_summary, indent=4), file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Get a summary of a data file.")
    parser.add_argument("--file-path", type=str, required=True, help="Path to the data file.")
    args = parser.parse_args()
    get_summary(args.file_path)
