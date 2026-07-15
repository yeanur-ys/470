from dataclasses import dataclass
import os


@dataclass(frozen=True)
class WorkerConfig:
    poll_interval_seconds: int
    neo4j_uri: str
    neo4j_user: str
    neo4j_password: str


def load_config() -> WorkerConfig:
    return WorkerConfig(
        poll_interval_seconds=int(os.getenv("POLL_INTERVAL_SECONDS", "30")),
        neo4j_uri=os.getenv("NEO4J_URI", "bolt://localhost:7687"),
        neo4j_user=os.getenv("NEO4J_USER", "neo4j"),
        neo4j_password=os.getenv("NEO4J_PASSWORD", "ngj_dev_password"),
    )
