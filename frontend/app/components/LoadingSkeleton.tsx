export function LoadingSkeleton() {
  return (
    <div className="skeleton-pulse space-y-4" aria-busy="true" aria-label="Loading content">
      <div className="h-8 bg-white/10 rounded-lg w-3/4"></div>
      <div className="h-64 bg-white/10 rounded-xl"></div>
      <div className="grid grid-cols-3 gap-4">
        <div className="h-20 bg-white/10 rounded-lg"></div>
        <div className="h-20 bg-white/10 rounded-lg"></div>
        <div className="h-20 bg-white/10 rounded-lg"></div>
      </div>
    </div>
  );
}
