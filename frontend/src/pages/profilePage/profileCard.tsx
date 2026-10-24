import { useAuth } from '../../context/AuthContext';
import '../../App.css';
import LogoutButton from './LogoutButton';

const columnLines = [
  [100, 86, 94, 72, 90, 64],
  [92, 78, 100, 68, 86, 74],
  [80, 96, 70, 90, 64, 100],
];

const Bars = ({ widths }: { widths: number[] }) => (
  <div className="ProfileSheet-col" aria-hidden="true">
    {widths.map((width, i) => (
      <span key={i} style={{ width: `${width}%` }} />
    ))}
  </div>
);

const ProfileCard = () => {
  const { user } = useAuth();

  const created = user?.created_at || user?.createdAt;
  const formattedCreatedDate = created
    ? new Date(created).toLocaleDateString('ru-RU', {
        day: 'numeric',
        month: 'long',
        year: 'numeric',
      })
    : '—';

  const companyId = user?.id != null ? String(user.id).padStart(4, '0') : '—';
  const companyName = user?.company_name || user?.name || 'Пользователь';

  return (
    <article className="ProfileCard ProfileSheet">
      <div className="ProfileSheet-paper" aria-hidden="true" />
      <div className="ProfileSheet-fold" aria-hidden="true" />

      <header className="ProfileSheet-mast">
        <span className="ProfileCard-kicker">деловая газета</span>
        <span className="ProfileSheet-issue">№ {companyId}</span>
        <LogoutButton compact />
      </header>

      <div className="ProfileSheet-spread">
        <section className="ProfileSheet-lead">
          <span className="ProfileCard-label-small">заголовок выпуска</span>
          <h1 className="ProfileCard-name">{companyName}</h1>
          <p className="ProfileSheet-dateline">
            <span>дата выпуска</span>
            {formattedCreatedDate}
          </p>
          <div className="ProfileSheet-desk">
            <Bars widths={columnLines[0]} />
          </div>
          <div className="ProfileSheet-phone">
            <Bars widths={columnLines[0].slice(0, 3)} />
          </div>
          <div className="ProfileSheet-block">очень важная информация</div>
        </section>

        <section className="ProfileSheet-copy">
          <div className="ProfileSheet-desk">
            <Bars widths={columnLines[1].slice(0, 4)} />
          </div>
          <div className="ProfileSheet-inset">
            <span>почта</span>
            <strong>{user?.email || '—'}</strong>
          </div>
          <div className="ProfileSheet-desk">
            <Bars widths={columnLines[1].slice(4, 6)} />
          </div>
          <div className="ProfileSheet-inset">
            <span>id</span>
            <strong className="ProfileCard-mono">{user?.id ?? '—'}</strong>
          </div>
          <div className="ProfileSheet-desk">
            <Bars widths={columnLines[2]} />
          </div>
          <div className="ProfileSheet-phone">
            <Bars widths={columnLines[1].slice(0, 2)} />
          </div>
        </section>
      </div>
    </article>
  );
};

export default ProfileCard;
