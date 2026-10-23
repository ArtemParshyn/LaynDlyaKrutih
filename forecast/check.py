from contract import ForecastRequest

MIN_HISTORY_DAYS = 30


class NotEnoughData(Exception):
    pass


def check(request: ForecastRequest) -> bool:
    if request.as_of is None or request.opening_balance is None:
        raise NotEnoughData("нужны дата и остаток")
    return len(request.history) >= MIN_HISTORY_DAYS
