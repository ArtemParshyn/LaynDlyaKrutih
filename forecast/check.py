from contract import ForecastRequest

MIN_HISTORY_DAYS = 30


class NotEnoughData(Exception):
    pass


def check(request: ForecastRequest) -> bool:
    if request.as_of is None or request.opening_balance is None:
        raise NotEnoughData("нужны дата и остаток")
    return len({row.date for row in request.history if row.date <= request.as_of}) >= MIN_HISTORY_DAYS
