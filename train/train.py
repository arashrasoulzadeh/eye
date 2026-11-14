"""Build a global bundle from reviewed Eye CSV exports."""
import argparse
from collections import Counter, defaultdict
import csv
import hashlib
import json
from pathlib import Path

HEADER = ["schema_version", "event_sequence", "time_gap_buckets", "purchased",
          "returned_within_7d", "prior_visits"]


def load_batches(folder):
    batches = {}
    catalog = Path(__file__).resolve().parents[1] / "internal/service/global_tags.go"
    allowed = set(catalog.read_text().split('"event": strings.Fields(`')[1].split("`")[0].split())
    allowed.update(("navigation", "engagement", "cart_add", "error", "other"))
    for path in sorted(folder.glob("*.csv")):
        with path.open(newline="", encoding="utf-8") as handle:
            reader = csv.reader(handle)
            if next(reader, None) != HEADER:
                raise ValueError(f"{path}: unsupported schema")
            rows = []
            for version, events, gaps, purchase, returned, visits in reader:
                sequence = tuple(events.split("|"))
                gap_values = gaps.split("|") if gaps else []
                if (version != "1" or any(e not in allowed for e in sequence)
                        or len(gap_values) != len(sequence) - 1
                        or any(gap not in ("0", "1", "2", "3") for gap in gap_values)
                        or purchase not in ("", "0", "1") or returned not in ("", "0", "1")
                        or visits not in ("", "0", "1", "2", "3", "4", "5")):
                    raise ValueError(f"{path}: invalid training row")
                rows.append((sequence, purchase, returned, visits))
        if rows:
            digest = hashlib.sha256(repr(sorted(rows)).encode()).hexdigest()
            batches[digest] = rows
    if not batches:
        raise ValueError("Provide at least one nonempty training CSV")
    return [batches[key] for key in sorted(batches)]


def split_batches(batches):
    if len(batches) > 1:
        return [row for batch in batches[:-1] for row in batch[:10000]], batches[-1][:10000]
    groups = defaultdict(list)
    for row in batches[0]:
        groups[row[0]].append(row)
    transition_groups = sum(len(sequence) > 1 for sequence in groups)
    if transition_groups < 2:
        raise ValueError("Single-file evaluation needs two distinct session sequences with transitions")
    sequences = sorted(groups, key=lambda sequence: (
        len(sequence) < 2, hashlib.sha256("|".join(sequence).encode()).digest()
    ))
    cutoff = min(max(1, len(sequences) // 5), transition_groups - 1)
    holdout = [row for sequence in sequences[:cutoff] for row in groups[sequence]]
    training = [row for sequence in sequences[cutoff:] for row in groups[sequence]]
    return training[:10000], holdout[:10000]


def fit(rows):
    transitions = defaultdict(Counter)
    purchases, returns = defaultdict(Counter), defaultdict(Counter)
    for sequence, purchase, returned, visits in rows:
        already_purchased = False
        for current, following in zip(sequence, sequence[1:]):
            transitions[current][following] += 1
            already_purchased |= current == "purchase_confirmed"
            if purchase and not already_purchased:
                purchases[current][purchase] += 1
        if returned and visits:
            returns[visits][returned] += 1

    def probabilities(counts):
        return {key: (values["1"] + 1) / (values.total() + 2)
                for key, values in counts.items()}

    return {"next": {key: counts.most_common(1)[0][0] for key, counts in transitions.items()},
            "conversion": probabilities(purchases), "return": probabilities(returns)}


def evaluate(bundle, rows):
    hits, examples = 0, 0
    for sequence, *_ in rows:
        for index in range(1, len(sequence)):
            context = "|".join(sequence[max(0, index - 3):index])
            prediction = bundle.get("sequence_next", {}).get(context)
            hits += (prediction or bundle["next"].get(sequence[index-1])) == sequence[index]
            examples += 1
    if not examples:
        raise ValueError("Holdout needs event transitions")
    return hits / examples


def outcome_metrics(bundle, rows):
    errors = {"conversion": [], "return": []}
    for sequence, purchased, returned, visits in rows:
        if purchased and len(sequence) > 1 and sequence[0] != "purchase_confirmed":
            probability = bundle["conversion"].get(sequence[0], 0.5)
            errors["conversion"].append((probability - int(purchased)) ** 2)
        if returned and visits:
            probability = bundle["return"].get(visits, 0.5)
            errors["return"].append((probability - int(returned)) ** 2)
    return {key: sum(values) / len(values) for key, values in errors.items() if values}


def temporal_candidate(rows):
    import torch
    from torch import nn

    torch.set_num_threads(2)
    torch.manual_seed(42)
    vocab = sorted({event for sequence, *_ in rows for event in sequence})
    indices = {event: index + 1 for index, event in enumerate(vocab)}
    counts = Counter()
    examples = []
    outcomes = []
    for sequence, *_ in rows:
        for index in range(1, len(sequence)):
            context = sequence[max(0, index - 3):index]
            counts[context] += 1
            examples.append([0] * (3-len(context)) + [indices[e] for e in context])
            outcomes.append(indices[sequence[index]] - 1)
    if not examples:
        return {}
    class Predictor(nn.Module):
        def __init__(self):
            super().__init__()
            self.embedding = nn.Embedding(len(vocab)+1, 16)
            self.layers = nn.Sequential(nn.Conv1d(16, 32, 2), nn.ReLU(), nn.Flatten(),
                                        nn.Linear(64, len(vocab)))

        def forward(self, tokens):
            return self.layers(self.embedding(tokens).transpose(1, 2))

    model = Predictor()
    optimizer = torch.optim.Adam(model.parameters())
    tokens, targets = torch.tensor(examples), torch.tensor(outcomes)
    for _ in range(5):
        for batch in torch.randperm(len(tokens)).split(512):
            optimizer.zero_grad()
            nn.functional.cross_entropy(model(tokens[batch]), targets[batch]).backward()
            optimizer.step()
    contexts = [context for context, count in counts.most_common(20000) if count >= 2]
    if not contexts:
        return {}
    encoded = [[0]*(3-len(context)) + [indices[e] for e in context] for context in contexts]
    with torch.no_grad():
        predicted = model(torch.tensor(encoded)).argmax(dim=1).tolist()
    return {"|".join(context): vocab[index] for context, index in zip(contexts, predicted)}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--data", type=Path, default=Path("train/data"))
    parser.add_argument("--version", required=True)
    parser.add_argument("--previous", type=Path)
    parser.add_argument("--cnn", action="store_true")
    args = parser.parse_args()
    batches = load_batches(args.data)
    training, holdout = split_batches(batches)
    bundle = fit(training)
    baseline = evaluate(bundle, holdout)
    accuracy = baseline
    if args.cnn:
        candidate = dict(bundle, sequence_next=temporal_candidate(training))
        score = evaluate(candidate, holdout)
        if score > accuracy:
            bundle, accuracy = candidate, score
    if not bundle["next"] or accuracy <= 0:
        raise ValueError("Candidate has no predictive coverage on holdout")
    outcomes = outcome_metrics(bundle, holdout)
    if args.previous and args.previous.exists():
        previous = json.loads(args.previous.read_text())
        previous_accuracy = evaluate(previous, holdout)
        previous_outcomes = outcome_metrics(previous, holdout)
        if accuracy < previous_accuracy or any(value > previous_outcomes[key] for key, value in outcomes.items()):
            raise ValueError("Candidate regressed on held-out predictions")
        if accuracy == previous_accuracy and outcomes == previous_outcomes:
            raise ValueError("Candidate did not improve held-out predictions")
    bundle.update(schema=1, version=args.version,
                  evaluation={"next_event_accuracy": accuracy, "transition_baseline": baseline,
                              "outcome_brier_scores": outcomes,
                              "training_sessions": len(training), "held_out_sessions": len(holdout)})
    Path("eye-global-model.json").write_text(json.dumps(bundle, sort_keys=True) + "\n")


if __name__ == "__main__":
    main()
