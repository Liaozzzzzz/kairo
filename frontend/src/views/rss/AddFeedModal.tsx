import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal, Input, Form, message, Button, Switch } from 'antd';
import { useRSSStore } from '@/store/useRSSStore';
import { BrowserOpenURL } from '@root/wailsjs/runtime/runtime';
import DownloadDir from '@/components/DownloadDir';
import { useSettingStore } from '@/store/useSettingStore';
import { useShallow } from 'zustand/react/shallow';
import { RSSFeed } from '@/types';

interface AddFeedModalProps {
  open: boolean;
  onClose: () => void;
  initialValues?: RSSFeed;
  mode?: 'add' | 'edit';
}

const AddFeedModal: React.FC<AddFeedModalProps> = ({
  open,
  onClose,
  initialValues,
  mode = 'add',
}) => {
  const { t } = useTranslation();
  const { defaultDir } = useSettingStore(
    useShallow((state) => ({
      defaultDir: state.defaultDir,
    }))
  );
  const [form] = Form.useForm();
  const { addFeed, updateFeed, isLoading } = useRSSStore();
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    if (open) {
      if (mode === 'edit' && initialValues) {
        form.setFieldsValue({
          url: initialValues.url,
          custom_dir: initialValues.custom_dir,
          download_latest: initialValues.download_latest,
          filters: initialValues.filters,
          tags: initialValues.tags,
          filename_template: initialValues.filename_template,
        });
      } else {
        form.resetFields();
        form.setFieldsValue({
          download_latest: true,
          custom_dir: defaultDir,
        });
      }
    }
  }, [open, mode, initialValues, form, defaultDir]);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setIsSubmitting(true);

      if (mode === 'edit' && initialValues) {
        await updateFeed({
          id: initialValues.id,
          custom_dir: values.custom_dir || '',
          download_latest: values.download_latest || false,
          filters: values.filters || '',
          tags: values.tags || '',
          filename_template: values.filename_template || '',
        });
        message.success(t('rss.modal.updateSuccess'));
      } else {
        await addFeed({
          url: values.url,
          custom_dir: values.custom_dir || '',
          download_latest: values.download_latest || false,
          filters: values.filters || '',
          tags: values.tags || '',
          filename_template: values.filename_template || '',
        });
        message.success(t('rss.modal.success'));
      }

      form.resetFields();
      onClose();
    } catch (error) {
      console.error(error);
      message.error(mode === 'edit' ? t('rss.modal.updateFailed') : t('rss.modal.failed'));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Modal
      title={mode === 'edit' ? t('rss.modal.editTitle') : t('rss.modal.title')}
      open={open}
      centered
      onCancel={onClose}
      onOk={handleSubmit}
      confirmLoading={isSubmitting || isLoading}
      destroyOnHidden
      width={600}
      okText={mode === 'edit' ? t('rss.modal.save') : t('rss.modal.add')}
      cancelText={t('rss.modal.cancel')}
    >
      <div className="mb-4 text-slate-500 text-sm">{t('rss.modal.desc')}</div>

      <Form
        form={form}
        layout="vertical"
        initialValues={{
          download_latest: true,
          custom_dir: defaultDir,
          url: 'http://192.168.31.65:12000/bilibili/popular/all',
        }}
      >
        <Form.Item
          name="url"
          label={t('rss.modal.url')}
          rules={[
            { required: true, message: t('rss.modal.urlRequired') },
            { type: 'url', message: t('rss.modal.urlInvalid') },
          ]}
          className="mb-2"
        >
          <Input
            placeholder="https://docs.rsshub.app/routes/youtube/user/@FKJ"
            disabled={mode === 'edit'}
          />
        </Form.Item>

        <div className="mb-3 p-2 flex items-center justify-between gap-2 bg-yellow-50 dark:bg-yellow-900/10 rounded border border-yellow-100 dark:border-yellow-900/20 text-xs">
          <span className="text-slate-600 dark:text-slate-400">{t('rss.modal.rsshubTip')}</span>
          <Button
            type="link"
            size="small"
            className="p-0 ml-1 h-auto"
            onClick={() => BrowserOpenURL('https://docs.rsshub.app/')}
          >
            {'>'}
          </Button>
        </div>

        <Form.Item label={t('rss.modal.customDir')} name="custom_dir" className="mb-4">
          <DownloadDir />
        </Form.Item>

        <Form.Item label={t('rss.modal.downloadLatest')} name="download_latest" className="mb-4">
          <Switch />
        </Form.Item>
        <Form.Item name="filters" label={t('rss.modal.filters')} className="mb-4">
          <Input placeholder={t('rss.modal.filtersPlaceholder')} />
        </Form.Item>
        <Form.Item name="tags" label={t('rss.modal.tags')} className="mb-4">
          <Input placeholder={t('rss.modal.tagsPlaceholder')} />
        </Form.Item>
        <Form.Item name="filename_template" label={t('rss.modal.template')} className="mb-4">
          <Input placeholder="%(uploader)s/%(title)s.%(ext)s" />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default AddFeedModal;
