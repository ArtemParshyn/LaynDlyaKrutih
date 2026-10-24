

interface SkeletonProps {
    className?: string;   // ← чтобы задать размеры СНАРУЖИ
    width?: string;       // ← опционально, если надо инлайн
    height?: string;
    circle?: boolean;     // ← для аватара
}

const Skeleton = ({ className = '', width, height, circle = false }: SkeletonProps) => {
    return (
        <div
            className={`skeleton ${circle ? 'skeleton-circle' : ''} ${className}`}
            style={{ width, height }}
        />
    );
};
export default Skeleton;