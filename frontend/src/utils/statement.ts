import { Transaction } from './csv';

const splitLine = (line: string, delimiter: string) => {
  const cells: string[] = [];
  let current = '';
  let quoted = false;
  for (const char of line) {
    if (char === '"') {
      quoted = !quoted;
      continue;
    }
    if (char === delimiter && !quoted) {
      cells.push(current.trim());
      current = '';
      continue;
    }
    current += char;
  }
  cells.push(current.trim());
  return cells;
};

const findHeader = (headers: string[], names: string[]) =>
  headers.findIndex((header) => names.some((name) => header.toLowerCase().includes(name)));

export const toBackendStatement = (text: string): string => {
  const clean = text.replace(/^\uFEFF/, '').trim();
  const lines = clean.split(/\r?\n/).filter((line) => line.trim());
  if (lines.length < 2) throw new Error('Файл пуст');

  const delimiter = lines[0].includes(';') ? ';' : ',';
  const headers = splitLine(lines[0], delimiter).map((header) => header.toLowerCase());
  const date = findHeader(headers, ['operation_date', 'дата', 'date']);
  const amount = findHeader(headers, ['amount', 'сумма']);
  const direction = findHeader(headers, ['direction', 'тип', 'type']);
  const category = findHeader(headers, ['category', 'категори']);
  const purpose = findHeader(headers, ['payment_purpose', 'назначени', 'описани', 'description', 'counterparty_name']);

  if (date < 0 || amount < 0) {
    throw new Error('В CSV нет даты и суммы');
  }

  const rows = ['Дата,Сумма,Тип,Категория,Описание'];
  lines.slice(1).forEach((line) => {
    const cells = splitLine(line, delimiter);
    if (!cells[date] || !cells[amount]) return;
    const kind = /списан|расход|out/i.test(cells[direction] || '') ? 'Расход' : 'Доход';
    rows.push([
      cells[date],
      cells[amount].replace(/\s/g, '').replace(',', '.'),
      kind,
      cells[category] || 'Прочее',
      `"${(cells[purpose] || '').replace(/"/g, '')}"`,
    ].join(','));
  });

  if (rows.length < 2) throw new Error('В файле нет операций');
  return rows.join('\n');
};

export const statementToTransactions = (statement: string): Transaction[] =>
  statement
    .split('\n')
    .slice(1)
    .map((line) => {
      const [date, amount, type, category, description = ''] = line.split(',');
      const value = Number(amount) || 0;
      return {
        date,
        amount: type === 'Расход' ? -Math.abs(value) : Math.abs(value),
        type: type === 'Расход' ? 'Расход' : 'Доход',
        category,
        description: description.replace(/"/g, ''),
      };
    });
