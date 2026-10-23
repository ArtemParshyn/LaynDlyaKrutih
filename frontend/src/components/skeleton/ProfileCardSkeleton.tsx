import  Skeleton  from './Skeleton';
import '../../App.css';

const ProfileCardSkeleton = () => {
    return (
        <div className="ProfileCard">
            <Skeleton circle className="skeleton-avatar" />
            <div className="ProfileCard-info">
                <Skeleton className="skeleton-title" />
                <Skeleton className="skeleton-text" />
                <Skeleton className="skeleton-text-short" />
            </div>
        </div>
    );
};

export default ProfileCardSkeleton;