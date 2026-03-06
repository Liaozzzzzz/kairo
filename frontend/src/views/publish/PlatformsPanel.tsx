import { useEffect, useState } from 'react';
import { Button, Form, Input, Modal, Space, Table, notification } from 'antd';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import { UpdatePublishPlatform } from '@root/wailsjs/go/main/App';
import { schema } from '@root/wailsjs/go/models';
import { usePublishStore } from '@/store/usePublishStore';

const PlatformsPanel = () => {
  const { t } = useTranslation();
  const { platforms, loadingPlatforms, fetchPlatforms } = usePublishStore();
  const [modalOpen, setModalOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [editing, setEditing] = useState<schema.PublishPlatform | null>(null);
  const [form] = Form.useForm();

  useEffect(() => {
    fetchPlatforms();
  }, []);

  const handleEdit = async (platform: schema.PublishPlatform) => {
    setEditing(platform);
    form.setFieldsValue({
      name: platform.name,
      display_name: platform.display_name,
    });
    setModalOpen(true);
  };

  const handleSave = async () => {
    if (!editing) return;
    try {
      const values = await form.validateFields();
      setSaving(true);
      await UpdatePublishPlatform(editing.id, values.display_name);
      notification.success({
        message: t('publish.platform.message.updateSuccess'),
      });
      setModalOpen(false);
      await fetchPlatforms();
    } catch (error) {
      notification.error({
        message: t('publish.platform.message.saveFailed'),
        description:
          error instanceof Error ? error.message : (error as string) || t('common.unknownError'),
      });
    } finally {
      setSaving(false);
    }
  };

  const columns = [
    {
      title: t('publish.platform.columns.displayName'),
      dataIndex: 'display_name',
      key: 'display_name',
      render: (_: string, record: schema.PublishPlatform) => record.display_name || record.name,
    },
    {
      title: t('publish.platform.columns.name'),
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: t('publish.platform.columns.updatedAt'),
      key: 'updated_at',
      render: (_: unknown, record: schema.PublishPlatform) =>
        dayjs(record.updated_at).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: t('publish.platform.columns.actions'),
      key: 'actions',
      render: (_: unknown, record: schema.PublishPlatform) => (
        <Space>
          <Button size="small" onClick={() => handleEdit(record)}>
            {t('common.edit')}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div className="flex flex-col gap-4">
      <Table
        rowKey="id"
        dataSource={platforms}
        columns={columns}
        loading={loadingPlatforms}
        pagination={false}
      />
      <Modal
        open={modalOpen}
        title={t('publish.platform.modal.editTitle')}
        onCancel={() => setModalOpen(false)}
        onOk={handleSave}
        confirmLoading={saving}
        okText={t('common.save')}
        cancelText={t('common.cancel')}
        destroyOnHidden
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label={t('publish.platform.columns.name')}>
            <Input disabled />
          </Form.Item>
          <Form.Item
            name="display_name"
            label={t('publish.platform.columns.displayName')}
            rules={[{ required: true, message: t('publish.platform.form.displayNameRequired') }]}
          >
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default PlatformsPanel;
