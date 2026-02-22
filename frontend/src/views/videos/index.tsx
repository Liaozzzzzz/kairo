import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useShallow } from 'zustand/react/shallow';
import { Input, Select } from 'antd';
import { SearchOutlined } from '@ant-design/icons';
import PageContainer from '@/components/PageContainer';
import PageHeader from '@/components/PageHeader';
import { Video } from '@/types';
import VideoList from './VideoList';
import VideoDetailModal from './VideoDetailModal';
import HighlightsModal from './HighlightsModal';
import { useVideoStore } from '@/store/useVideoStore';

export default function Videos() {
  const { t } = useTranslation();
  const { videos, loading, fetchVideos } = useVideoStore(
    useShallow((state) => ({
      videos: state.videos,
      loading: state.loading,
      fetchVideos: state.fetchVideos,
    }))
  );

  const [searchQuery, setSearchQuery] = useState('');
  const [filterStatus, setFilterStatus] = useState('all');
  const [selectedVideoId, setSelectedVideoId] = useState<string | null>(null);
  const [highlightVideoId, setHighlightVideoId] = useState<string | null>(null);

  useEffect(() => {
    fetchVideos(filterStatus, searchQuery);
  }, [searchQuery, filterStatus, fetchVideos]);

  const handleSelectVideo = (video: Video) => {
    setSelectedVideoId(video.id);
  };

  const handleRefresh = () => {
    fetchVideos(filterStatus, searchQuery);
  };

  const headerContent = (
    <div className="flex items-center gap-4">
      <Input
        prefix={<SearchOutlined className="text-slate-400" />}
        placeholder={t('videos.search_placeholder')}
        className="w-64"
        value={searchQuery}
        onChange={(e) => setSearchQuery(e.target.value)}
        allowClear
      />
      <Select
        value={filterStatus}
        onChange={setFilterStatus}
        options={[
          { value: 'all', label: t('videos.filter_all') },
          { value: 'analyzed', label: t('videos.filter_analyzed') },
          { value: 'unanalyzed', label: t('videos.filter_unanalyzed') },
        ]}
        className="w-32"
      />
    </div>
  );

  return (
    <PageContainer
      viewClass="px-10"
      header={
        <PageHeader
          title={t('app.sidebar.videos')}
          subtitle={t('videos.description')}
          extra={headerContent}
        />
      }
    >
      <VideoList
        videos={videos}
        loading={loading}
        onSelect={handleSelectVideo}
        onRefresh={handleRefresh}
        onHighlights={(video) => setHighlightVideoId(video.id)}
      />

      {selectedVideoId && (
        <VideoDetailModal
          videoId={selectedVideoId}
          isOpen={!!selectedVideoId}
          onClose={() => setSelectedVideoId(null)}
        />
      )}

      {highlightVideoId && (
        <HighlightsModal
          videoId={highlightVideoId}
          isOpen={!!highlightVideoId}
          onClose={() => setHighlightVideoId(null)}
        />
      )}
    </PageContainer>
  );
}
