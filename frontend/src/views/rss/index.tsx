import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useRSSStore } from '@/store/useRSSStore';
import { PlusOutlined } from '@ant-design/icons';
import FeedList from './FeedList';
import FeedItems from './FeedItems';
import RSSWelcome from './RSSWelcome';
import AddFeedModal from './AddFeedModal';
import PageContainer from '@/components/PageContainer';
import PageHeader from '@/components/PageHeader';
import { RSSFeed } from '@/types';

const RSSView: React.FC = () => {
  const { t } = useTranslation();
  const { fetchFeeds, selectedFeedId } = useRSSStore();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingFeed, setEditingFeed] = useState<RSSFeed | undefined>(undefined);

  useEffect(() => {
    fetchFeeds();
  }, [fetchFeeds]);

  const handleEdit = (feed: RSSFeed) => {
    setEditingFeed(feed);
    setIsModalOpen(true);
  };

  const handleClose = () => {
    setIsModalOpen(false);
    setEditingFeed(undefined);
  };

  return (
    <PageContainer
      viewClass="px-10"
      header={
        <div className="flex flex-col gap-6">
          <PageHeader title={t('rss.title')} subtitle={t('rss.subtitle')} />
          <div className="flex items-center gap-3 border-b border-slate-200 dark:border-neutral-800 pb-4">
            <FeedList onEdit={handleEdit} />
            <div
              className="flex flex-col items-center gap-1.5 cursor-pointer group shrink-0 px-1"
              onClick={() => {
                setEditingFeed(undefined);
                setIsModalOpen(true);
              }}
            >
              <div className="w-14 h-14 rounded-full border-2 border-dashed border-slate-300 dark:border-neutral-700 flex items-center justify-center text-slate-400 group-hover:border-primary group-hover:text-primary transition-colors bg-slate-50 dark:bg-neutral-900">
                <PlusOutlined className="text-xl" />
              </div>
              <span className="text-[11px] text-slate-500 group-hover:text-primary transition-colors">
                {t('rss.add')}
              </span>
            </div>
          </div>
        </div>
      }
    >
      {selectedFeedId ? <FeedItems /> : <RSSWelcome />}
      <AddFeedModal
        open={isModalOpen}
        onClose={handleClose}
        initialValues={editingFeed}
        mode={editingFeed ? 'edit' : 'add'}
      />
    </PageContainer>
  );
};

export default RSSView;
