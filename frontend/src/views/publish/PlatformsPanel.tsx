import { forwardRef, useEffect, useImperativeHandle, useState } from 'react';
import {
  Button,
  Form,
  Input,
  Modal,
  Space,
  Table,
  notification,
  Select,
  Alert,
  Typography,
} from 'antd';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import { CreatePublishPlatform, UpdatePublishPlatform } from '@root/wailsjs/go/main/App';
import { schema } from '@root/wailsjs/go/models';
import { usePublishStore } from '@/store/usePublishStore';

const { Text, Paragraph } = Typography;

export type PlatformsPanelRef = {
  addPlatform: () => void;
};

export default forwardRef<PlatformsPanelRef, unknown>(function PlatformsPanel(_, ref) {
  const { t } = useTranslation();
  const { platforms, loadingPlatforms, fetchPlatforms } = usePublishStore();
  const [modalOpen, setModalOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [editing, setEditing] = useState<schema.PublishPlatform | null>(null);
  const [form] = Form.useForm();
  const typeValue = Form.useWatch('type', form);

  useEffect(() => {
    fetchPlatforms();
  }, []);

  const addPlatform = () => {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({
      type: 'openclaw',
    });
    setModalOpen(true);
  };

  const handleEdit = async (platform: schema.PublishPlatform) => {
    setEditing(platform);
    form.setFieldsValue({
      name: platform.name,
      display_name: platform.display_name,
      type: platform.type || 'builtin',
    });
    setModalOpen(true);
  };

  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setSaving(true);

      if (editing) {
        await UpdatePublishPlatform(editing.id, values.display_name);
        notification.success({
          message: t('publish.platform.message.updateSuccess'),
        });
      } else {
        await CreatePublishPlatform(values.name, values.display_name, values.type);
        notification.success({
          message: t('publish.platform.message.createSuccess'),
        });
      }

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
      title: t('publish.platform.columns.type'),
      dataIndex: 'type',
      key: 'type',
      render: (type: string) =>
        type === 'openclaw'
          ? t('publish.platform.type.openclaw')
          : t('publish.platform.type.builtin'),
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

  useImperativeHandle(ref, () => ({
    addPlatform: () => addPlatform(),
  }));

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
        title={
          editing ? t('publish.platform.modal.editTitle') : t('publish.platform.modal.createTitle')
        }
        onCancel={() => setModalOpen(false)}
        onOk={handleSave}
        confirmLoading={saving}
        okText={t('common.save')}
        cancelText={t('common.cancel')}
        destroyOnHidden
        centered
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="type"
            label={t('publish.platform.form.type')}
            rules={[{ required: true }]}
          >
            <Select
              disabled={true}
              options={[
                { label: t('publish.platform.type.builtin'), value: 'builtin' },
                { label: t('publish.platform.type.openclaw'), value: 'openclaw' },
              ]}
            />
          </Form.Item>

          <Form.Item
            name="name"
            label={t('publish.platform.columns.name')}
            rules={[{ required: true, message: t('publish.platform.form.nameRequired') }]}
            extra={
              typeValue === 'openclaw' ? t('publish.platform.form.openClawNameExtra') : undefined
            }
          >
            <Input disabled={!!editing} placeholder="e.g. xiaohongshu-ops" />
          </Form.Item>

          <Form.Item
            name="display_name"
            label={t('publish.platform.columns.displayName')}
            rules={[{ required: true, message: t('publish.platform.form.displayNameRequired') }]}
          >
            <Input />
          </Form.Item>

          {typeValue === 'openclaw' && (
            <div className="mt-4">
              <Alert
                message={t('publish.platform.form.openClawParams')}
                description={
                  <div className="text-xs font-mono max-h-60 overflow-y-auto">
                    <Paragraph className="mb-2">
                      {t('publish.platform.form.openClawDesc')}
                      <Text code className="block mt-1">
                        openclaw run &lt;name&gt; --input &lt;json&gt;
                      </Text>
                    </Paragraph>

                    <Text strong>{t('publish.platform.form.uploadInput')}</Text>
                    <pre className="bg-gray-100 dark:bg-gray-800 p-2 rounded mt-1 mb-2">
                      {`{
  "command": "upload",
  "video_path": "/absolute/path/to/video.mp4",
  "title": "Video Title",
  "description": "Video Description",
  "tags": ["tag1", "tag2"],
  "cookie_path": "/path/to/cookies.json"
}`}
                    </pre>

                    <Text strong>{t('publish.platform.form.validateInput')}</Text>
                    <pre className="bg-gray-100 dark:bg-gray-800 p-2 rounded mt-1">
                      {`{
  "command": "validate",
  "cookie_path": "/path/to/cookies.json"
}`}
                    </pre>
                  </div>
                }
                type="info"
                showIcon
              />
            </div>
          )}
        </Form>
      </Modal>
    </div>
  );
});
