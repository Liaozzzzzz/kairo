import { useEffect, useMemo, useState } from 'react';
import { Grid, CellComponentProps } from 'react-window';
import { useTranslation } from 'react-i18next';
import { Button, Checkbox, Divider, Image } from 'antd';
import { models } from '@root/wailsjs/go/models';
import { ImageFallback } from '@/data/variables';
import { formatDuration } from '@/lib/timer';
import { DownloadOutlined, PlayCircleOutlined } from '@ant-design/icons';
import DownloadDir from '@/components/DownloadDir';
import { useShallow } from 'zustand/react/shallow';
import { useSettingStore } from '@/store/useSettingStore';

interface VideoResultProps {
  videoInfo: models.VideoInfo;
  onStartDownload: ({ newDir, playList }: { newDir: string; playList?: number[] }) => void;
}

const PlaylistResult = ({ videoInfo, onStartDownload }: VideoResultProps) => {
  const { t } = useTranslation();

  const defaultDir = useSettingStore(useShallow((state) => state.defaultDir));

  const [newDir, setNewDir] = useState(defaultDir);
  const [selectedPlaylistItems, setSelectedPlaylistItems] = useState<number[]>([]);

  const columns = useMemo(() => {
    if (!Array.isArray(videoInfo.playlist_items)) {
      return [];
    }
    const result: Array<[models.PlaylistItem, models.PlaylistItem | undefined]> = [];
    for (let i = 0; i < videoInfo.playlist_items.length; i += 2) {
      result.push([videoInfo.playlist_items[i], videoInfo.playlist_items[i + 1]]);
    }
    return result;
  }, [videoInfo.playlist_items]);

  const onToggleAll = (checked: boolean) => {
    if (checked) {
      setSelectedPlaylistItems(videoInfo.playlist_items.map((item) => item.index));
    } else {
      setSelectedPlaylistItems([]);
    }
  };

  useEffect(() => {
    if (videoInfo.playlist_items?.length) {
      setSelectedPlaylistItems(videoInfo.playlist_items.map((item) => item.index));
    } else {
      setSelectedPlaylistItems([]);
    }
  }, [videoInfo]);

  const renderCard = (item: models.PlaylistItem) => {
    return (
      <div className="h-full bg-white border border-slate-200 rounded-xl overflow-hidden shadow-sm h-full">
        <div className="relative">
          {item.thumbnail ? (
            <Image
              src={item.thumbnail}
              referrerPolicy="no-referrer"
              alt=""
              width="100%"
              height="88px"
              fallback={ImageFallback}
            />
          ) : (
            <div className="w-full h-[88px] flex items-center justify-center text-gray-300">
              <PlayCircleOutlined className="text-2xl" />
            </div>
          )}
          <div className="absolute top-1 left-1 bg-white/90 rounded-md px-1 py-0.5">
            <Checkbox
              checked={selectedPlaylistItems.includes(item.index)}
              onChange={(e) =>
                setSelectedPlaylistItems((prev) =>
                  e.target.checked ? [...prev, item.index] : prev.filter((i) => i !== item.index)
                )
              }
            />
          </div>
        </div>
        <div className="px-2">
          <div
            className="text-sm font-medium leading-[1.3] overflow-hidden  line-clamp-2"
            title={item.title}
          >
            {item.title}
          </div>
          <div className="text-xs text-muted-foreground mt-1 flex items-center gap-2">
            <span>{t('downloads.playlistIndex', { index: item.index })}</span>
            <span>{formatDuration(item.duration)}</span>
          </div>
        </div>
      </div>
    );
  };

  const renderCell = ({
    columnIndex,
    rowIndex,
    style,
    columns: cellColumns,
    columnWidth: cellColumnWidth,
    rowHeight: cellRowHeight,
    renderCard: cellRenderCard,
    ariaAttributes,
  }: CellComponentProps<{
    columns: Array<[models.PlaylistItem, models.PlaylistItem | undefined]>;
    columnWidth: number;
    rowHeight: number;
    renderCard: (item: models.PlaylistItem) => JSX.Element;
  }>) => {
    const item = cellColumns[columnIndex]?.[rowIndex];
    if (!item) return null;
    return (
      <div
        {...ariaAttributes}
        style={{
          ...style,
          width: cellColumnWidth,
          height: cellRowHeight,
        }}
      >
        {cellRenderCard(item)}
      </div>
    );
  };

  return (
    <div className="space-y-3 animate-in fade-in slide-in-from-top-2 duration-300 delay-50">
      <div className="space-y-1">
        <div className="flex items-center justify-between gap-2">
          <span className="inline-flex items-center rounded-full bg-slate-200 px-2 py-0.5 text-xs font-medium text-slate-600 shrink-0">
            {t('downloads.playlistItems')}
          </span>
          {videoInfo.title && (
            <div className="text-sm font-semibold text-foreground truncate" title={videoInfo.title}>
              {videoInfo.title}
            </div>
          )}
          <Checkbox
            indeterminate={
              selectedPlaylistItems.length > 0 &&
              selectedPlaylistItems.length < videoInfo.playlist_items.length
            }
            checked={
              selectedPlaylistItems.length > 0 &&
              selectedPlaylistItems.length === videoInfo.playlist_items.length
            }
            className="shrink-0 ml-4"
            onChange={(e) => onToggleAll(e.target.checked)}
          >
            {t('downloads.selectAll')}
          </Checkbox>
        </div>
      </div>
      <Grid
        columnCount={columns.length}
        rowCount={2}
        cellComponent={renderCell}
        cellProps={{
          columns,
          renderCard,
          columnWidth: 240,
          rowHeight: 160,
        }}
        columnWidth={250}
        rowHeight={170}
        overscanCount={2}
      />
      <Divider size="large" />
      <div className="flex items-end justify-between gap-4">
        <DownloadDir defaultDir={newDir} setNewDir={setNewDir} />
        <Button
          type="primary"
          onClick={() => onStartDownload({ newDir, playList: selectedPlaylistItems })}
          disabled={selectedPlaylistItems.length === 0}
          icon={<DownloadOutlined />}
        >
          {t('downloads.start')}
        </Button>
      </div>
    </div>
  );
};

export default PlaylistResult;
