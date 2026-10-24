import Skeleton from './Skeleton';
import '../../App.css';

const ProfileCardSkeleton = () => {
  return (
    <div className="ProfileCard ProfileSheet ProfileCard--skeleton" aria-hidden="true">
      <div className="ProfileSheet-mast">
        <Skeleton className="skeleton-text-short" />
        <Skeleton className="skeleton-circle ProfileCard-skeleton-dot" />
      </div>
      <div className="ProfileSheet-spread">
        <div>
          <Skeleton className="skeleton-title" />
          <Skeleton className="skeleton-text" />
          <Skeleton className="skeleton-text" />
        </div>
        <div className="ProfileSheet-columns">
          <div className="ProfileSheet-col">
            <Skeleton className="skeleton-text" />
            <Skeleton className="skeleton-text" />
            <Skeleton className="skeleton-text-short" />
          </div>
        </div>
      </div>
    </div>
  );
};

export default ProfileCardSkeleton;
