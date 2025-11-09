"""Add question name and cancellation flag

Revision ID: b63ae0f0f5b0
Revises: ebcc4f8f4dc4
Create Date: 2025-11-09 13:10:00.000000
"""

from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa

# revision identifiers, used by Alembic.
revision: str = "b63ae0f0f5b0"
down_revision: Union[str, Sequence[str], None] = "ebcc4f8f4dc4"
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    op.add_column("question", sa.Column("name", sa.String(length=50), nullable=True))
    op.add_column(
        "question",
        sa.Column(
            "is_cancelled",
            sa.Boolean(),
            nullable=False,
            server_default=sa.text("0"),
        ),
    )

    question_table = sa.table(
        "question",
        sa.column("id", sa.Integer()),
        sa.column("exam_id", sa.Integer()),
        sa.column("name", sa.String(length=50)),
    )
    bind = op.get_bind()
    result = bind.execute(
        sa.select(question_table.c.id, question_table.c.exam_id).order_by(
            question_table.c.exam_id, question_table.c.id
        )
    )

    current_exam_id = None
    position = 0
    for row in result:
        if row.exam_id != current_exam_id:
            current_exam_id = row.exam_id
            position = 1
        else:
            position += 1

        bind.execute(
            question_table.update()
            .where(question_table.c.id == row.id)
            .values(name=str(position))
        )

    op.alter_column("question", "name", existing_type=sa.String(length=50), nullable=False)
    op.alter_column(
        "question",
        "is_cancelled",
        server_default=None,
    )


def downgrade() -> None:
    op.drop_column("question", "is_cancelled")
    op.drop_column("question", "name")
