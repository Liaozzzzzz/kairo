import { useState } from 'react';
import { Header } from './Header';
import { TaskList } from './TaskList';
import { AddTaskModal } from './AddTaskModal';
import { LogModal } from './LogModal';
import { Scrollbar } from '@/components/Scrollbar';

export default function Downloads() {
  const [isAddOpen, setIsAddOpen] = useState(false);
  const [viewLogId, setViewLogId] = useState<string | null>(null);
  const [filter, setFilter] = useState('downloading');

  return (
    <div className="h-full w-full bg-background text-foreground flex flex-col items-center overflow-hidden">
      <div className="w-full max-w-5xl h-full py-10 ">
        <Scrollbar
          className="h-full"
          viewClass="px-10"
          header={
            <div className="mb-8 px-10">
              <Header
                onOpenAdd={() => setIsAddOpen(true)}
                filter={filter}
                onFilterChange={setFilter}
              />
            </div>
          }
        >
          <TaskList onViewLog={setViewLogId} filter={filter} />
        </Scrollbar>
      </div>

      <AddTaskModal
        isOpen={isAddOpen}
        onClose={() => setIsAddOpen(false)}
        onSuccess={() => setFilter('downloading')}
      />
      <LogModal viewLogId={viewLogId} onClose={() => setViewLogId(null)} />
    </div>
  );
}
