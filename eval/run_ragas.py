import json
import math
import os
import httpx
import numpy as np
import pandas as pd
from dotenv import load_dotenv

load_dotenv()
from datetime import datetime
from datasets import Dataset
from ragas import evaluate
from ragas.metrics import faithfulness, context_precision, context_recall
from langchain_openai import ChatOpenAI
from ragas.llms import LangchainLLMWrapper


class GroqChatOpenAI(ChatOpenAI):
    """ChatOpenAI pointed at Groq's OpenAI-compatible endpoint.
    Forces finish_reason='stop' so RAGAS 0.2.6 doesn't raise
    LLMDidNotFinishException on Groq's 'eos_token' / 'length' responses.
    """
    def _generate(self, messages, stop=None, run_manager=None, **kwargs):
        result = super()._generate(messages, stop=stop, run_manager=run_manager, **kwargs)
        for gen_list in result.generations:
            for gen in gen_list:
                if gen.generation_info:
                    gen.generation_info["finish_reason"] = "stop"
        return result

API_URL = os.getenv("API_URL", "http://localhost:8080/api/chat")
GROQ_API_KEY = os.getenv("GROQ_API_KEY", "")
GROQ_MODEL = os.getenv("GROQ_MODEL", "llama-3.1-8b-instant")
DATASET_FILE = os.getenv("DATASET_FILE", "eval_dataset.json")

METRIC_NAMES = ["faithfulness", "answer_relevancy", "context_precision", "context_recall"]

SCORE_EMOJI = {
    "good":    "✅",
    "medium":  "⚠️",
    "bad":     "❌",
}

def score_label(val: float) -> str:
    if val >= 0.75:
        return SCORE_EMOJI["good"]
    if val >= 0.5:
        return SCORE_EMOJI["medium"]
    return SCORE_EMOJI["bad"]


def call_api(question: str) -> dict:
    resp = httpx.post(API_URL, json={"query": question, "language": "id"}, timeout=600)
    resp.raise_for_status()
    data = resp.json()
    return {
        "answer": data["answer"],
        "contexts": [s["text"] for s in (data.get("sources") or [])],
    }


def build_report(golden: list, df) -> str:
    lines = []
    ts = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    lines.append("=" * 70)
    lines.append(f"  UNILA AI — RAGAS Evaluation Report")
    lines.append(f"  Generated : {ts}")
    lines.append(f"  Model     : {GROQ_MODEL}")
    lines.append(f"  Dataset   : {DATASET_FILE}  ({len(golden)} questions)")
    lines.append("=" * 70)

    # Per-question detail
    lines.append("\n📋 PER-QUESTION SCORES\n")
    for i, row in df.iterrows():
        item = golden[i]
        lines.append(f"[{i+1:02d}] {item['question']}")
        lines.append(f"     Source : {item.get('source', '-')}")
        lines.append(f"     Answer : {str(row.get('answer', ''))[:120].strip()}...")

        score_parts = []
        for m in METRIC_NAMES:
            val = row.get(m)
            if val is None:
                score_parts.append(f"{m}: N/A")
            else:
                score_parts.append(f"{m}: {val:.2f} {score_label(val)}")
        lines.append("     Scores : " + " | ".join(score_parts))
        lines.append("")

    # Aggregate
    lines.append("=" * 70)
    lines.append("📊 AGGREGATE SCORES\n")
    agg = {}
    for m in METRIC_NAMES:
        col = df.get(m)
        if col is not None:
            agg[m] = col.mean()

    col_w = max(len(m) for m in METRIC_NAMES) + 2
    for m, val in agg.items():
        bar_len = int(val * 30)
        bar = "█" * bar_len + "░" * (30 - bar_len)
        lines.append(f"  {m:<{col_w}} {val:.4f}  [{bar}] {score_label(val)}")

    overall = sum(agg.values()) / len(agg) if agg else 0
    lines.append(f"\n  {'Overall Average':<{col_w}} {overall:.4f}  {score_label(overall)}")
    lines.append("")

    # Per-source breakdown
    sources = {}
    for i, row in df.iterrows():
        src = golden[i].get("source", "unknown")
        sources.setdefault(src, []).append(i)

    if len(sources) > 1:
        lines.append("=" * 70)
        lines.append("📁 SCORES BY SOURCE DOCUMENT\n")
        for src, idxs in sources.items():
            sub = df.iloc[idxs]
            src_scores = []
            for m in METRIC_NAMES:
                col = sub.get(m)
                if col is not None:
                    src_scores.append(f"{m}: {col.mean():.2f}")
            lines.append(f"  {src}")
            lines.append(f"    {' | '.join(src_scores)}")
            lines.append("")

    lines.append("=" * 70)
    return "\n".join(lines)


CACHE_FILE = "responses_cache.json"
SCORES_CACHE_FILE = "scores_cache.json"

def load_cache() -> dict:
    if os.path.exists(CACHE_FILE):
        with open(CACHE_FILE) as f:
            return json.load(f)
    return {}

def save_cache(cache: dict):
    with open(CACHE_FILE, "w") as f:
        json.dump(cache, f, ensure_ascii=False, indent=2)

def load_scores_cache() -> dict:
    if os.path.exists(SCORES_CACHE_FILE):
        with open(SCORES_CACHE_FILE) as f:
            return json.load(f)
    return {}

def save_scores_cache(scores: dict):
    with open(SCORES_CACHE_FILE, "w") as f:
        json.dump(scores, f, ensure_ascii=False, indent=2)


def main():
    with open(DATASET_FILE) as f:
        golden = json.load(f)

    cache = load_cache()
    questions, answers, contexts, ground_truths = [], [], [], []

    print(f"\nQuerying {len(golden)} questions against {API_URL}...\n")
    for i, item in enumerate(golden, 1):
        q = item["question"]
        if q in cache:
            print(f"  [{i:02d}/{len(golden)}] (cached) {q[:60]}...")
            result = cache[q]
        else:
            print(f"  [{i:02d}/{len(golden)}] {q[:65]}...")
            result = call_api(q)
            cache[q] = result
            save_cache(cache)

        questions.append(q)
        ground_truths.append(item["ground_truth"])
        answers.append(result["answer"])
        contexts.append(result["contexts"] if result["contexts"] else [""])

    dataset = Dataset.from_dict({
        "question": questions,
        "answer": answers,
        "contexts": contexts,
        "ground_truth": ground_truths,
    })

    llm = LangchainLLMWrapper(
        GroqChatOpenAI(
            api_key=GROQ_API_KEY,
            base_url="https://api.groq.com/openai/v1",
            model=GROQ_MODEL,
            max_tokens=2048,
        ),
        is_finished_parser=lambda _: True,
    )

    scores_cache = load_scores_cache()

    def needs_eval(q, metric):
        s = scores_cache.get(q, {}).get(metric)
        return s is None or (isinstance(s, float) and math.isnan(s))

    # Phase 1: context metrics
    ctx_questions = [q for q in questions if needs_eval(q, "context_precision") or needs_eval(q, "context_recall")]
    if ctx_questions:
        idx = [questions.index(q) for q in ctx_questions]
        ds1 = Dataset.from_dict({k: [v for i, v in enumerate([questions, answers, contexts, ground_truths][j]) if i in idx]
                                  for j, k in enumerate(["question","answer","contexts","ground_truth"])})
        print(f"\n[1/2] Running context metrics for {len(ctx_questions)} questions...\n")
        r1 = evaluate(ds1, metrics=[context_precision, context_recall], llm=llm, raise_exceptions=False)
        for i, q in enumerate(ctx_questions):
            scores_cache.setdefault(q, {})
            scores_cache[q]["context_precision"] = float(r1.to_pandas()["context_precision"].iloc[i])
            scores_cache[q]["context_recall"] = float(r1.to_pandas()["context_recall"].iloc[i])
        save_scores_cache(scores_cache)
    else:
        print("\n[1/2] context metrics: all cached ✓\n")

    # Phase 2: faithfulness
    faith_questions = [q for q in questions if needs_eval(q, "faithfulness")]
    if faith_questions:
        idx = [questions.index(q) for q in faith_questions]
        ds2 = Dataset.from_dict({k: [v for i, v in enumerate([questions, answers, contexts, ground_truths][j]) if i in idx]
                                  for j, k in enumerate(["question","answer","contexts","ground_truth"])})
        print(f"\n[2/2] Running faithfulness for {len(faith_questions)} questions...\n")
        r2 = evaluate(ds2, metrics=[faithfulness], llm=llm, raise_exceptions=False)
        for i, q in enumerate(faith_questions):
            scores_cache.setdefault(q, {})
            scores_cache[q]["faithfulness"] = float(r2.to_pandas()["faithfulness"].iloc[i])
        save_scores_cache(scores_cache)
    else:
        print("\n[2/2] faithfulness: all cached ✓\n")

    # Build final df from scores_cache
    rows = []
    for q in questions:
        s = scores_cache.get(q, {})
        rows.append({
            "question": q,
            "faithfulness": s.get("faithfulness", np.nan),
            "context_precision": s.get("context_precision", np.nan),
            "context_recall": s.get("context_recall", np.nan),
        })
    df = pd.DataFrame(rows)

    # Save raw CSV
    csv_path = f"eval_results_{datetime.now().strftime('%Y%m%d_%H%M%S')}.csv"
    df.to_csv(csv_path, index=False)

    # Build & save report
    report = build_report(golden, df)
    report_path = csv_path.replace(".csv", ".txt")
    with open(report_path, "w") as f:
        f.write(report)

    print(report)
    print(f"\nFiles saved:")
    print(f"  CSV    → {csv_path}")
    print(f"  Report → {report_path}")


if __name__ == "__main__":
    main()
