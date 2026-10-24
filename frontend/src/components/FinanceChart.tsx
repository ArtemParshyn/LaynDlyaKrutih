import { AreaChart, Area, XAxis, YAxis, CartesianGrid, ResponsiveContainer } from 'recharts';
import '../App.css';

interface ChartPoint {
    date: string;
    balance: number;
}

interface FinanceChartProps {
    data: ChartPoint[];
}

const FinanceChart = ({ data }: FinanceChartProps) => {
    return (
        <div className="FinanceChart">
            <h3 className="SectionTitle">Прогноз остатка</h3>

            <ResponsiveContainer width="100%" height={300}>
                <AreaChart data={data} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
                    <defs>
                        <linearGradient id="colorBalance" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="var(--accent)" stopOpacity={0.3}/>
                            <stop offset="95%" stopColor="var(--accent)" stopOpacity={0}/>
                        </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" stroke="var(--border-soft)" vertical={false} />
                    <XAxis dataKey="date" stroke="var(--text-muted)" />
                    <YAxis stroke="var(--text-muted)" />
                    <Area type="monotone" dataKey="balance" stroke="var(--accent)" strokeWidth={3} fill="url(#colorBalance)" />
                </AreaChart>
            </ResponsiveContainer>
        </div>
    );
};

export default FinanceChart;