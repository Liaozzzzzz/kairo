import { useState } from 'react';
import { Header } from './Header';
import { TaskList } from './TaskList';
import { LogModal } from './LogModal';
import PageContainer from '@/components/PageContainer';

export default function Tasks() {
  const [viewLogId, setViewLogId] = useState<string | null>(null);
  const [filter, setFilter] = useState('downloading');

  return (
    <PageContainer viewClass="px-10" header={<Header filter={filter} onFilterChange={setFilter} />}>
      <TaskList onViewLog={setViewLogId} filter={filter} />
      <LogModal viewLogId={viewLogId} onClose={() => setViewLogId(null)} />
    </PageContainer>
  );
}
