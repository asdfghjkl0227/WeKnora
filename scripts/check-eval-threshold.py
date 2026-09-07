#!/usr/bin/env python3
"""读取评测结果 JSON，逐项和阈值比较，任一指标低于阈值则退出码为 1。

读取文件（可用环境变量覆盖）：
  EVALUATION_RESULT_FILE    评测结果 JSON（默认 evaluation-result.json）
  EVALUATION_THRESHOLD_FILE 阈值 JSON（默认 scripts/eval-thresholds.json）
"""
import json
import os
import sys

RESULT_FILE = os.environ.get("EVALUATION_RESULT_FILE", "evaluation-result.json")
THRESHOLD_FILE = os.environ.get("EVALUATION_THRESHOLD_FILE", "scripts/eval-thresholds.json")

with open(RESULT_FILE, encoding="utf-8") as f:
    result = json.load(f)
with open(THRESHOLD_FILE, encoding="utf-8") as f:
    thresholds = json.load(f)

# 检索指标 + 生成指标合并成一个平铺字典
retrieval = result.get("retrieval_metrics") or {}
generation = result.get("generation_metrics") or {}
metrics = {**retrieval, **generation}

failed = False
print("=" * 54)
print(f"{'指标':<14}{'实际值':>10}{'期望阈值':>12}   结果")
print("-" * 54)
for name, expected in sorted(thresholds.items()):
    actual = float(metrics.get(name, 0.0))
    ok = actual >= expected
    if not ok:
        failed = True
    mark = "通过" if ok else "退化"
    print(f"{name:<14}{actual:>10.4f}{expected:>12}   {mark}")
print("=" * 54)

if failed:
    print("评测门禁失败：有指标低于阈值")
    sys.exit(1)
print("评测门禁通过：所有指标均达到阈值")
