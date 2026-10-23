import '../App.css';
import { useReactTable, getCoreRowModel, flexRender, ColumnDef } from '@tanstack/react-table';
import { useMemo } from 'react';

//1 строка из СSV
export interface Transaction {
    date: string;
    amount: number;
    category: string;
    counterparty: string;
    description?: string;
}

// 2. Описываем, как будет выглядеть сгруппированная строка (для таблицы)
interface GroupedTransaction {
    month: string;
    category: string;
    counterparty: string;
    amount: number;
}

interface AnalyzedTableProps {
    transactions: Transaction[];
}



const columns: ColumnDef<GroupedTransaction>[] = [
    {accessorKey:'month', header:'Месяц'},
    { accessorKey: 'category', header:'Категория'},
    {accessorKey:'counterparty', header:'Контрагент'},
    {accessorKey:'amount', header:'Сумма',
        cell: (info) => {
            const val = info.getValue() as number;
            // Красим приход в зеленый, расход в красный
            return <span style={{ color: val > 0 ? 'green' : 'red', fontWeight: 'bold' }}>
                {val > 0 ? '+' : ''}{val.toLocaleString('ru-RU')} ₽
            </span>;
        }}


];

const AnalyzedTable = ({transactions}:AnalyzedTableProps ) => {
    const groupedData = useMemo(() => {
        if (!transactions || transactions.length === 0) return [];

        const months = ['Январь', 'Февраль', 'Март', 'Апрель', 'Май', 'Июнь', 'Июль', 'Август', 'Сентябрь', 'Октябрь', 'Ноябрь', 'Декабрь'];
        const groups: Record<string, GroupedTransaction> = {};
        transactions.forEach(t => {
            const dateObj = new Date(t.date);
            const monthName = months[dateObj.getMonth()];

            // Создаем уникальный ключ для группы: "Октябрь_Продукты_Пятерочка"
            const groupKey = `${monthName}_${t.category}_${t.counterparty}`;

            if (!groups[groupKey]) {
                groups[groupKey] = {
                    month: monthName,
                    category: t.category,
                    counterparty: t.counterparty,
                    amount: 0,
                };
            }

            // Прибавляем сумму к группе
            groups[groupKey].amount += t.amount;
        });

        // Превращаем объект обратно в массив
        return Object.values(groups);
    }, [transactions]);
    // 5. Создаем экземпляр таблицы
    const table = useReactTable<GroupedTransaction>({
        data: groupedData,
        columns,
        getCoreRowModel: getCoreRowModel(),
    });

    return (
        <div style={{ width: '100%', overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left' }}>
                <thead>
                {table.getHeaderGroups().map((headerGroup) => (
                    <tr key={headerGroup.id} style={{ borderBottom: '2px solid var(--border-strong)' }}>
                        {headerGroup.headers.map((header) => (
                            <th key={header.id} style={{ padding: '12px 8px', color: 'var(--text-muted)' }}>
                                {flexRender(header.column.columnDef.header, header.getContext())}
                            </th>
                        ))}
                    </tr>
                ))}
                </thead>
                <tbody>
                {table.getRowModel().rows.length === 0 ? (
                    <tr>
                        <td colSpan={columns.length} style={{ textAlign: 'center', padding: '20px', color: 'var(--text-faint)' }}>
                            Нет данных для отображения
                        </td>
                    </tr>
                ) : (
                    table.getRowModel().rows.map((row) => (
                        <tr key={row.id} style={{ borderBottom: '1px solid var(--border)' }}>
                            {row.getVisibleCells().map((cell) => (
                                <td key={cell.id} style={{ padding: '10px 8px', color: 'var(--text)' }}>
                                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                                </td>
                            ))}
                        </tr>
                    ))
                )}
                </tbody>
            </table>
        </div>
    );
};


export default AnalyzedTable;