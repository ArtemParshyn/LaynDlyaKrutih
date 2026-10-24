import os
from pathlib import Path

BASE_DIR = Path(__file__).resolve().parent.parent
ARTIFACTS_DIR = BASE_DIR / "artifacts"
UPLOADS_DIR = ARTIFACTS_DIR / "uploads"

# Prophet предупреждает, если годовой сезонности меньше двух полных циклов.
MIN_DAYS_YEARLY = 730
# Короче этого ряда Prophet ненадёжен — обучение отклоняем.
MIN_HISTORY_DAYS = 60
# Сколько последних дней отдаём под holdout при подборе гиперпараметров.
HOLDOUT_DAYS = 30
# Optuna: экран + сколько лучших конфигов гонять на rolling CV.
N_TRIALS = int(os.getenv("N_TRIALS", "20"))
TOP_K = int(os.getenv("TOP_K", "3"))
RANDOM_SEED = 42

# Сколько компаний обучаем одновременно: Prophet+Optuna упираются в CPU,
# поэтому лишние задачи ждут в статусе queued.
MAX_CONCURRENT_TRAININGS = 2

# Предел размера загружаемой выписки. 50 МБ — с запасом больше любой
# выписки из кейса, но защищает от случайной загрузки гигабайта.
MAX_UPLOAD_BYTES = 50 * 1024 * 1024

# Вероятность кассового разрыва: доля ширины интервала Prophet,
# которая лежит ниже нуля. Ниже порога риск не показываем.
GAP_PROBABILITY_THRESHOLD = 0.05
