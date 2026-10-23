import { useState, useEffect } from 'react';
import { getAllUsers, UserInfo } from '../api/auth';
import { useAuth } from '../context/AuthContext';
import { CircleLoader } from 'react-spinners';
import { Link } from 'react-router-dom';
import '../App.css';

const SettingsPage = () => {
  const { user } = useAuth();
  const [users, setUsers] = useState<UserInfo[]>([]);
  const [loadingUsers, setLoadingUsers] = useState(false);
  const [error, setError] = useState('');

  const fetchUsers = async () => {
    setLoadingUsers(true);
    setError('');
    try {
      const data = await getAllUsers();
      setUsers(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось загрузить список пользователей');
    } finally {
      setLoadingUsers(false);
    }
  };

  useEffect(() => {
    fetchUsers();
  }, []);

  return (
    <div style={{ maxWidth: '800px', margin: '0 auto', padding: '20px' }}>
      <h1 className="PageTitle">Настройки и система</h1>

      <div style={{
        background: 'var(--surface)',
        borderRadius: '16px',
        padding: '24px',
        marginBottom: '24px',
        border: '1px solid var(--border)'
      }}>
        <h3 style={{ marginTop: 0 }}>Текущий аккаунт</h3>
        <p><strong>Компания:</strong> {user?.company_name || user?.name}</p>
        <p><strong>Email:</strong> {user?.email}</p>
        <p><strong>ID в системе:</strong> #{user?.id}</p>
        <Link to="/profile" className="Button" style={{ display: 'inline-block', marginTop: '10px' }}>
          Управление профилем
        </Link>
      </div>

      <div style={{
        background: 'var(--surface)',
        borderRadius: '16px',
        padding: '24px',
        border: '1px solid var(--border)'
      }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
          <h3 style={{ margin: 0 }}>Список зарегистрированных организаций (API /users)</h3>
          <button className="Button" onClick={fetchUsers} disabled={loadingUsers}>
            {loadingUsers ? <CircleLoader size={16} color="#fff" /> : 'Обновить'}
          </button>
        </div>

        {error && <div className="error-message">{error}</div>}

        {loadingUsers ? (
          <div style={{ display: 'flex', justifyContent: 'center', padding: '20px' }}>
            <CircleLoader size={30} color="var(--accent)" />
          </div>
        ) : users.length === 0 ? (
          <p style={{ color: 'var(--text-muted)' }}>Организации не найдены</p>
        ) : (
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left' }}>
              <thead>
                <tr style={{ borderBottom: '1px solid var(--border)', color: 'var(--text-muted)', fontSize: '14px' }}>
                  <th style={{ padding: '8px' }}>ID</th>
                  <th style={{ padding: '8px' }}>Компания</th>
                  <th style={{ padding: '8px' }}>Email</th>
                  <th style={{ padding: '8px' }}>Дата создания</th>
                </tr>
              </thead>
              <tbody>
                {users.map((u) => (
                  <tr key={u.id} style={{ borderBottom: '1px solid var(--border)', fontSize: '14px' }}>
                    <td style={{ padding: '8px' }}>#{u.id}</td>
                    <td style={{ padding: '8px', fontWeight: u.id === user?.id ? 'bold' : 'normal' }}>
                      {u.company_name} {u.id === user?.id && '(Вы)'}
                    </td>
                    <td style={{ padding: '8px' }}>{u.email}</td>
                    <td style={{ padding: '8px', color: 'var(--text-muted)' }}>
                      {u.created_at ? new Date(u.created_at).toLocaleDateString('ru-RU') : '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
};

export default SettingsPage;