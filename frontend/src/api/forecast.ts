import { apiClient } from './apiClient';
import { API_URL } from './config';

export interface ForecastDay {
  date: string;
  expected_balance: number;
  balance_low: number;
  balance_high: number;
  inflow: number;
  outflow: number;
}

export interface ForecastAlert {
  date: string;
  shortfall: number;
  chance: number;
  reason: string;
}

export interface RarePayment {
  amount: number;
  is_income: boolean;
  seen: number;
  roughly_every_days: number;
}

// Совпадает с ledger/dto.ScheduledPayment и gateway/dto.ScheduledPayment.
// amount — копейки. Категорию и признак дохода считает бэкенд, фронт их не выводит.
export interface ScheduledPayment {
  date: string;
  amount: number;
  is_income: boolean;
  category: string;
}

export interface StatementOperation {
  date: string;
  amount: number;
  category: string;
  description: string;
  is_income: boolean;
}

export interface BalancePoint {
  date: string;
  balance: number;
}

export interface ForecastResponse {
  days: ForecastDay[];
  alerts: ForecastAlert[];
  rough: boolean;
  recurring_payments: ScheduledPayment[];
  operations?: StatementOperation[];
  history?: BalancePoint[];
  rare_payments?: RarePayment[];
}

export const uploadStatement = (file: File) => {
  const body = new FormData();
  body.append('file', file);
  return apiClient<ForecastResponse>(`${API_URL}/api/statements`, {
    method: 'POST',
    body,
  });
};

export const uploadStatementText = (csv: string, filename = 'statement.csv') => {
  const file = new File([csv], filename, { type: 'text/csv' });
  return uploadStatement(file);
};

export const getForecast = () =>
  apiClient<ForecastResponse>(`${API_URL}/api/forecast`, { method: 'GET' });

export const createEvent = (event: {
  date: string;
  amount: number;
  is_income: boolean;
  category: string;
}) =>
  apiClient<ForecastResponse>(`${API_URL}/api/events`, {
    method: 'POST',
    body: JSON.stringify(event),
  });

export const calculateScenario = (events: Array<{
  date: string;
  amount: number;
  is_income: boolean;
  category: string;
}>) =>
  apiClient<ForecastResponse>(`${API_URL}/api/scenarios`, {
    method: 'POST',
    body: JSON.stringify({ events }),
  });

export const rubles = (kopecks: number) => Math.round(kopecks) / 100;
