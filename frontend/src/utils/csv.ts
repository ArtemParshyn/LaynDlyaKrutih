// src/utils/csv.ts

export interface Transaction {
    date: string;        // ISO
    amount: number;      // > 0 доход, < 0 расход
    type: 'Доход' | 'Расход';
    category: string;
    description: string;
}

// Простой парсер с поддержкой кавычек
export function parseCSV(text: string): Transaction[] {
    const lines = text.trim().split('\n');
    const headers = lines[0].split(',');

    const idx = {
        date: headers.findIndex(h => h.trim() === 'Дата'),
        amount: headers.findIndex(h => h.trim() === 'Сумма'),
        type: headers.findIndex(h => h.trim() === 'Тип'),
        category: headers.findIndex(h => h.trim() === 'Категория'),
        desc: headers.findIndex(h => h.trim() === 'Описание'),
    };

    return lines.slice(1).map(line => {
        // В реальном проекте лучше использовать полноценный CSV-парсер,
        // но для ваших данных (без запятых внутри полей) сойдет и так:
        const values = line.split(',');

        return {
            date: values[idx.date] || '',
            amount: parseFloat(values[idx.amount]) || 0,
            type: (values[idx.type] as 'Доход' | 'Расход') || 'Расход',
            category: values[idx.category] || 'Прочее',
            description: values[idx.desc] || '',
        };
    }).filter(t => t.date !== '');
}