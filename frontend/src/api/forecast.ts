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
  manual?: boolean;
}

export interface BalancePoint {
  date: string;
  balance: number;
}

export interface TrainingStatus {
  status: 'queued' | 'running' | 'succeeded' | 'failed' | string;
  error?: string;
}

export interface ForecastResponse {
  days: ForecastDay[];
  alerts: ForecastAlert[];
  rough: boolean;
  recurring_payments: ScheduledPayment[];
  operations?: StatementOperation[];
  history?: BalancePoint[];
  rare_payments?: RarePayment[];
  training?: TrainingStatus;
  summary?: string;
}

const pulseUrl = (pulseId: string, path: string) =>
  `${API_URL}/api/pulses/${pulseId}${path}`;

export const uploadStatement = (pulseId: string, file: File, append = false) => {
  const body = new FormData();
  body.append('file', file);
  const url = append
    ? pulseUrl(pulseId, '/statements?mode=append')
    : pulseUrl(pulseId, '/statements');
  return apiClient<ForecastResponse>(url, {
    method: 'POST',
    body,
  });
};

export const uploadStatementText = (pulseId: string, csv: string, filename = 'statement.csv') => {
  const file = new File([csv], filename, { type: 'text/csv' });
  return uploadStatement(pulseId, file);
};

export const getForecast = (pulseId: string) =>
  apiClient<ForecastResponse>(pulseUrl(pulseId, '/forecast'), { method: 'GET' });

export const processForecast = (pulseId: string) =>
  apiClient<ForecastResponse>(pulseUrl(pulseId, '/forecast/process'), { method: 'POST' });

export const addOperation = (pulseId: string, operation: {
  date: string;
  amount: number;
  is_income: boolean;
  description?: string;
  recurring?: boolean;
}) =>
  apiClient<ForecastResponse>(pulseUrl(pulseId, '/operations'), {
    method: 'POST',
    body: JSON.stringify({ ...operation, category: 'Прочее' }),
  });

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
