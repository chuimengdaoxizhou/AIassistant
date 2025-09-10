# backend/python/extract_facts.py

# To install LangExtract, run:
# pip install langextract

import argparse
import json
import sys

from langextract import Processor, llms
from pydantic import BaseModel, Field

# 1. Define the schema for the data to be extracted
class PersonalPreference(BaseModel):
    """个人偏好（喜好、厌恶）"""
    preference: str = Field(..., description="用户的个人偏好，例如喜欢的食物、产品、活动、娱乐等")

class PersonalDetail(BaseModel):
    """重要个人详情（姓名、关系、重要日期）"""
    detail: str = Field(..., description="用户的重要个人详情，例如姓名、家庭关系、朋友关系、重要的纪念日等")

class PlanAndIntent(BaseModel):
    """计划和意图"""
    plan: str = Field(..., description="用户的计划和意图，例如即将到来的事件、旅行、目标等")

class ActivityPreference(BaseModel):
    """活动和服务偏好"""
    preference: str = Field(..., description="用户的活动和服务偏好，例如餐饮、旅行、爱好等方面的偏好")

class HealthPreference(BaseModel):
    """健康和健身偏好"""
    preference: str = Field(..., description="用户的健康和健身偏好，例如饮食限制、健身习惯等")

class ProfessionalDetail(BaseModel):
    """职业详情"""
    detail: str = Field(..., description="用户的职业详情，例如工作职位、工作习惯、职业目标等")

class MiscellaneousInfo(BaseModel):
    """其他杂项信息"""
    info: str = Field(..., description="用户的其他杂项信息，例如喜欢的书籍、电影、品牌等")

class ExtractedFacts(BaseModel):
    facts: list[
        PersonalPreference | PersonalDetail | PlanAndIntent | ActivityPreference | HealthPreference | ProfessionalDetail | MiscellaneousInfo
    ]

# 2. Create a function to extract facts
def extract_facts(text: str) -> dict:
    """Extracts facts from text using LangExtract."""
    # Configure the LLM. This example uses Gemini, but you can also use other models.
    # The API key should be set as an environment variable for security.
    # os.environ["GEMINI_API_KEY"] = "YOUR_API_KEY"
    llm = llms.Gemini()

    # Create a Processor with the schema and LLM
    processor = Processor(schema=ExtractedFacts, llm=llm)

    # Process the text
    extracted_data = processor.process(text)

    return extracted_data.model_dump()

# 3. Main execution block
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Extract facts from text using LangExtract.")
    parser.add_argument("text", type=str, help="The text to extract facts from.")
    args = parser.parse_args()

    try:
        facts = extract_facts(args.text)
        # Print the extracted facts as a JSON string to standard output
        json.dump(facts, sys.stdout)
    except Exception as e:
        # Print error to stderr
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)
