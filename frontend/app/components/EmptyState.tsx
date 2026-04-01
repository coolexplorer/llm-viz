interface Props {
  icon: React.ReactNode;
  title: string;
  description: string;
}

export function EmptyState({ icon, title, description }: Props) {
  return (
    <div className="flex flex-col items-center justify-center h-64 text-slate-500" role="status">
      <div className="w-16 h-16 mb-4 text-slate-600">
        {icon}
      </div>
      <h3 className="text-lg font-semibold text-slate-400 mb-2">{title}</h3>
      <p className="text-sm text-slate-500 text-center max-w-md">{description}</p>
    </div>
  );
}
